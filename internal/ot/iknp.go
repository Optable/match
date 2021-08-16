package ot

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/cipher"
	"github.com/optable/match/internal/util"
)

/*
1 out of 2 IKNP OT extension
from the paper: Extending Oblivious Transfers Efficiently
by Yushal Ishai, Joe Kilian, Kobbi Nissim, and Erez Petrank in 2003.
reference: https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf

A possible improvement is to use bitset to store the bit matrices/bit sets.
*/

const (
	iknpCurve      = "P256"
	iknpCipherMode = cipher.XORBlake3
)

type iknp struct {
	baseOT OT
	m      int
	k      int
	msgLen []int
	prng   *rand.Rand
}

func NewIKNP(m, k, baseOT int, ristretto bool, msgLen []int) (iknp, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = m
	}

	ot, err := NewBaseOT(baseOT, ristretto, k, iknpCurve, baseMsgLen, iknpCipherMode)
	if err != nil {
		return iknp{}, err
	}

	return iknp{baseOT: ot, m: m, k: k, msgLen: msgLen, prng: rand.New(rand.NewSource(time.Now().UnixNano()))}, nil
}

func (ext iknp) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// sample choice bits for baseOT
	s := make([]uint8, ext.k)
	if err = util.SampleBitSlice(ext.prng, s); err != nil {
		return err
	}

	// act as receiver in baseOT to receive q^j
	q := make([][]uint8, ext.k)
	if err = ext.baseOT.Receive(s, q, rw); err != nil {
		return err
	}

	// transpose q to m x k matrix for easier row operations
	q = util.Transpose(q)

	var key, ciphertext []byte
	// encrypt messages and send them
	for i := range messages {
		for choice, plaintext := range messages[i] {
			key = q[i]
			if choice == 1 {
				key, err = util.XorBytes(q[i], s)
				if err != nil {
					return err
				}
			}

			ciphertext, err = cipher.Encrypt(iknpCipherMode, key, uint8(choice), plaintext)
			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// send ciphertext
			if _, err = rw.Write(ciphertext); err != nil {
				return err
			}
		}
	}

	return
}

func (ext iknp) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// Sample k x m matrix T
	t, err := util.SampleRandomBitMatrix(ext.prng, ext.k, ext.m)
	if err != nil {
		return err
	}

	// make k pairs of m bytes baseOT messages: {t^j, t^j xor choices}
	baseMsgs := make([][][]byte, ext.k)
	for j := range baseMsgs {
		baseMsgs[j] = make([][]byte, 2)
		// []uint8 = []byte, since byte is an alias to uint8
		baseMsgs[j][0] = t[j]
		baseMsgs[j][1], err = util.XorBytes(t[j], choices)
		if err != nil {
			return err
		}
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return err
	}

	// compute k x m transpose to access columns easier
	t = util.Transpose(t)

	e := make([][]byte, 2)
	for i := range choices {
		// compute # of bytes to be read
		l := cipher.EncryptLen(iknpCipherMode, ext.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// decrypt received ciphertext using key (choices[i], t_i)
		messages[i], err = cipher.Decrypt(iknpCipherMode, t[i], choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("error decrypting sender messages: %s", err)
		}
	}

	return
}
