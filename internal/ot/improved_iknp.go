package ot

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

/*
1 out of 2 IKNP OT extension
from the paper: Extending Oblivious Transfers Efficiently
by Yushal Ishai, Joe Kilian, Kobbi Nissim, and Erez Petrank in 2003.
reference: https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf
*/

type imprvIknp struct {
	baseOT OT
	m      int
	k      int
	msgLen []int
	prng   *rand.Rand
	g      *blake3.Hasher
}

func NewImprovedIknp(m, k, baseOt int, ristretto bool, msgLen []int) (imprvIknp, error) {
	// send k columns of messages of length k
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k
	}

	ot, err := NewBaseOT(baseOt, ristretto, k, iknpCurve, baseMsgLen, iknpCipherMode)
	if err != nil {
		return imprvIknp{}, err
	}
	g := blake3.New()

	return imprvIknp{baseOT: ot, m: m, k: k, msgLen: msgLen,
		prng: rand.New(rand.NewSource(time.Now().UnixNano())),
		g:    g}, nil
}

func (ext imprvIknp) Send(messages [][2][]byte, rw io.ReadWriter) (err error) {
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
		q[i], err = xorCipherWithPRG(ext.g, seeds[i], e[s[i]])
	}

	//fmt.Printf("q:\n%v\n", q)
	q = util.Transpose(q)

	// encrypt messages and send them
	var key, ciphertext []byte
	for i := range messages {
		for choice, plaintext := range messages[i] {
			key = q[i]
			if choice == 1 {
				key, err = xorBytes(q[i], s)
				if err != nil {
					return err
				}
			}

			ciphertext, err = encrypt(iknpCipherMode, key, uint8(choice), plaintext)
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

func (ext imprvIknp) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
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
	baseMsgs := make([][2][]byte, ext.k)
	for j := range baseMsgs {
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
		u, err = xorBytes(t[row], choices)
		if err != nil {
			return err
		}

		maskedTj, err := xorCipherWithPRG(ext.g, baseMsgs[row][0], t[row])
		if err != nil {
			return err
		}

		maskedUj, err := xorCipherWithPRG(ext.g, baseMsgs[row][1], u)
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
		l := encryptLen(iknpCipherMode, ext.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// decrypt received ciphertext using key (choices[i], t_i)
		messages[i], err = decrypt(iknpCipherMode, t[i], choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("error decrypting sender messages: %s", err)
		}
	}

	return
}
