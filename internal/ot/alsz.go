package ot

import (
	"crypto/rand"
	"fmt"
	"io"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

/*
1 out of 2 IKNP OT extension
from the paper: Extending Oblivious Transfers Efficiently
by Yushal Ishai, Joe Kilian, Kobbi Nissim, and Erez Petrank in 2003.
reference: https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf

The improvement from normal IKNP is to use pseudorandom generators
to send short OT messages instead of the full encrypted messages.
*/

type alsz struct {
	baseOT OT
	m      int
	k      int
	msgLen []int
	drbg   int
}

func NewAlsz(m, k, baseOt, drbg int, ristretto bool, msgLen []int) (alsz, error) {
	// send k columns of messages of length k
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k
	}

	ot, err := NewBaseOT(baseOt, ristretto, k, iknpCurve, baseMsgLen, iknpCipherMode)
	if err != nil {
		return alsz{}, err
	}

	return alsz{baseOT: ot, m: m, k: k, drbg: drbg, msgLen: msgLen}, nil
}

func (ext alsz) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// sample choice bits for baseOT
	s := make([]uint8, ext.k)
	if err = util.SampleBitSlice(rand.Reader, s); err != nil {
		return err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]uint8, ext.k)
	if err = ext.baseOT.Receive(s, seeds, rw); err != nil {
		return err
	}

	// receive masked columns u
	u := make([]byte, ext.m)
	q := make([][]byte, ext.k)
	for col := range q {
		if _, err = io.ReadFull(rw, u); err != nil {
			return err
		}

		q[col], err = crypto.PseudorandomGenerate(ext.drbg, seeds[col], ext.m)
		if err != nil {
			return err
		}

		q[col], err = util.XorBytes(util.AndByte(s[col], u), q[col])
		if err != nil {
			return err
		}
	}

	q = util.Transpose(q)

	// encrypt messages and send them
	var key, ciphertext []byte
	for i := range messages {
		for choice, plaintext := range messages[i] {
			key = q[i]
			if choice == 1 {
				key, err = util.XorBytes(key, s)
				if err != nil {
					return err
				}
			}

			ciphertext, err = crypto.Encrypt(iknpCipherMode, key, uint8(choice), plaintext)
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

func (ext alsz) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// sample k x k bit mtrix
	seeds, err := util.SampleRandomBitMatrix(rand.Reader, 2*ext.k, ext.k)
	if err != nil {
		return err
	}

	baseMsgs := make([][][]byte, ext.k)
	for j := range baseMsgs {
		baseMsgs[j] = make([][]byte, 2)
		baseMsgs[j][0] = seeds[2*j]
		baseMsgs[j][1] = seeds[2*j+1]
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return err
	}

	var t = make([][]byte, ext.k)
	var u = make([]byte, ext.m)
	// u^i = G(seeds[1])
	// t^i = d^i ^ u^i
	for col := range t {
		t[col], err = crypto.PseudorandomGenerate(ext.drbg, baseMsgs[col][0], ext.m)
		if err != nil {
			return err
		}

		u, err = crypto.PseudorandomGenerate(ext.drbg, baseMsgs[col][1], ext.m)
		if err != nil {
			return err
		}

		u, err = util.XorBytes(u, t[col])
		if err != nil {
			return err
		}

		u, err = util.XorBytes(u, choices)
		if err != nil {
			return err
		}

		// send u^col
		if _, err = rw.Write(u); err != nil {
			return err
		}
	}

	// transpose t to m x k for easier row operations
	t = util.Transpose(t)

	// receive encrypted messages.
	e := make([][]byte, 2)
	for i := range choices {
		// compute nb of bytes to be read
		l := crypto.EncryptLen(iknpCipherMode, ext.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// decrypt received ciphertext using key (choices[i], t_i)
		messages[i], err = crypto.Decrypt(iknpCipherMode, t[i], choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("error decrypting sender messages: %s", err)
		}
	}

	return
}
