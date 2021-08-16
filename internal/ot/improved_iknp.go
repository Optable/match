package ot

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/cipher"
	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

/*
1 out of 2 IKNP OT extension
from the paper: Extending Oblivious Transfers Efficiently
by Yushal Ishai, Joe Kilian, Kobbi Nissim, and Erez Petrank in 2003.
reference: https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf

The improvement from normal IKNP is just to use pseudorandom generators
to send short OT messages instead of the full encrypted messages.
Computation overhead seems to make it more time consuming at the expense
of the smaller network costs.
*/

type imprvIKNP struct {
	baseOT OT
	m      int
	k      int
	msgLen []int
	prng   *rand.Rand
	g      *blake3.Hasher
}

func NewImprovedIKNP(m, k, baseOt int, ristretto bool, msgLen []int) (imprvIKNP, error) {
	// send k columns of messages of length k
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k
	}

	ot, err := NewBaseOT(baseOt, ristretto, k, iknpCurve, baseMsgLen, iknpCipherMode)
	if err != nil {
		return imprvIKNP{}, err
	}
	g := blake3.New()

	return imprvIKNP{baseOT: ot, m: m, k: k, msgLen: msgLen,
		prng: rand.New(rand.NewSource(time.Now().UnixNano())),
		g:    g}, nil
}

func (ext imprvIKNP) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// sample choice bits for baseOT
	s := make([]uint8, ext.k)
	if err = util.SampleBitSlice(ext.prng, s); err != nil {
		return err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]uint8, ext.k)
	if err = ext.baseOT.Receive(s, seeds, rw); err != nil {
		return err
	}

	// receive masked columns
	q := make([][]byte, ext.k)
	e := make([][]byte, 2)
	for i := range q {
		// read both msg
		for j := range e {
			// each column is m bytes long
			e[j] = make([]byte, ext.m)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}
		// unmask e and store it in q.
		q[i], err = cipher.XorCipherWithPRG(ext.g, seeds[i], e[s[i]])
	}

	//fmt.Printf("q:\n%v\n", q)
	q = util.Transpose(q)

	// encrypt messages and send them
	var key, ciphertext []byte
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

func (ext imprvIKNP) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// compute actual messages to be sent
	// t is pseudorandom binary matrix
	t, err := util.SampleRandomBitMatrix(ext.prng, ext.k, ext.m)
	if err != nil {
		return err
	}

	// make k pairs of m bytes baseOT messages: {t^j, t^j xor choices}
	baseMsgs := make([][][]byte, ext.k)
	for j := range baseMsgs {
		baseMsgs[j] = make([][]byte, 2)
		for b := range baseMsgs[j] {
			baseMsgs[j][b] = make([]byte, ext.k)
			err = util.SampleBitSlice(ext.prng, baseMsgs[j][b])
			if err != nil {
				return err
			}
		}
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return err
	}

	// compute actual t^j, u^j and send them masked: t^j xor G(s)
	// u^j = t^j xor choice
	var u []uint8
	//u := make([][]uint8, ext.k)
	for row := range t {
		//u, err = xorBytes(t[row], choices)
		u, err = util.XorBytes(t[row], choices)
		if err != nil {
			return err
		}

		maskedTj, err := cipher.XorCipherWithPRG(ext.g, baseMsgs[row][0], t[row])
		if err != nil {
			return err
		}

		maskedUj, err := cipher.XorCipherWithPRG(ext.g, baseMsgs[row][1], u)
		if err != nil {
			return err
		}

		// send t^j
		if _, err = rw.Write(maskedTj); err != nil {
			return err
		}

		// send u^j
		if _, err = rw.Write(maskedUj); err != nil {
			return err
		}
	}

	t = util.Transpose(t)

	// receive encrypted messages.
	e := make([][]byte, 2)
	for i := range choices {
		// compute nb of bytes to be read
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
