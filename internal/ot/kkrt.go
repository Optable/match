package ot

/*
1 out of N OT extension
based on IKNP 1 out of 2 OT extension
from the paper: Improved OT extension for Transferring Short Secrets
by Vladimir Kolesnikov and Ranjit Kumaresan in 2013, and later
from the paper: Efficient Batched Oblivious PRF with Applications to Private Set Intersection
by Vladimir Kolesnikov, Ranjit Kumaresan, Mike Rosulek, and Ni Treu in 2016.
Reference:	https://eprint.iacr.org/2013/491.pdf (Improved IKNP)
			http://dx.doi.org/10.1145/2976749.2978381 (KKRT)
*/

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/util"
)

const (
	kkrtCurve      = "P256"
	kkrtCipherMode = XORBlake3
)

type kkrt struct {
	baseOT OT
	m      int
	k      int
	n      int
	msgLen []int
	prng   *rand.Rand
}

func NewKKRT(m, k, n, baseOT int, ristretto bool, msgLen []int) (kkrt, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = m
	}

	ot, err := NewBaseOT(baseOT, ristretto, k, kkrtCurve, baseMsgLen, kkrtCipherMode)
	if err != nil {
		return kkrt{}, err
	}

	return kkrt{baseOT: ot, m: m, k: k, n: n, msgLen: msgLen, prng: rand.New(rand.NewSource(time.Now().UnixNano()))}, nil
}

func (ext kkrt) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]uint8, 16)
	if _, err = ext.prng.Read(sk); err != nil {
		return nil
	}

	// send the secret key
	if _, err := rw.Write(sk); err != nil {
		return err
	}

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

	var key, x, ciphertext []byte
	// encrypt messages and send them
	for i := range messages {
		// proof of concept, suppose we have n messages, and the choice string is an integer in [1, ..., n]
		for choice, plaintext := range messages[i] {
			// compute q_i ^ (C(r) & s)
			x, _ = util.AndBytes(pseudorandomCode(sk, ext.k, []byte{byte(choice)}), s)
			key, _ = util.XorBytes(q[i], x)

			ciphertext, err = encrypt(kkrtCipherMode, key, uint8(choice), plaintext)
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

func (ext kkrt) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	sk := make([]byte, 16)
	if _, err = io.ReadFull(rw, sk); err != nil {
		return err
	}

	// Sample m x k matrix T
	t, err := util.SampleRandomBitMatrix(ext.prng, ext.m, ext.k)
	if err != nil {
		return err
	}

	// make m pairs of k bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][][]byte, ext.m)
	for i := range baseMsgs {
		baseMsgs[i] = make([][]byte, 2)
		// |t[i]| = k
		baseMsgs[i][0] = t[i]
		// do we need mod 2 for each bytes from the return of pseudorandomCode?
		baseMsgs[i][1], err = util.XorBytes(t[i], pseudorandomCode(sk, ext.k, []byte{choices[i]}))
		if err != nil {
			return err
		}
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(util.Transpose3D(baseMsgs), rw); err != nil {
		return err
	}

	e := make([][]byte, ext.n)
	for i := range choices {
		// compute # of bytes to be read
		l := encryptLen(kkrtCipherMode, ext.msgLen[i])
		// read all msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// decrypt received ciphertext using key (choices[i], t_i)
		messages[i], err = decrypt(kkrtCipherMode, t[i], choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("error decrypting sender messages: %s", err)
		}
	}

	return
}
