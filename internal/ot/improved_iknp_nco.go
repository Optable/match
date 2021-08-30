package ot

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/crypto"
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

type imprvIKNPNCO struct {
	baseOT OT
	m      int
	k      int
	n      int
	msgLen []int
	prng   *rand.Rand
	g      *blake3.Hasher
}

func NewImprovedIKNPNCO(m, k, n, baseOt int, ristretto bool, msgLen []int) (imprvIKNPNCO, error) {
	// send k columns of messages of length k
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k
	}

	ot, err := NewBaseOT(baseOt, ristretto, k, iknpCurve, baseMsgLen, iknpCipherMode)
	if err != nil {
		return imprvIKNPNCO{}, err
	}
	g := blake3.New()

	return imprvIKNPNCO{baseOT: ot, m: m, k: k, n: n, msgLen: msgLen,
		prng: rand.New(rand.NewSource(time.Now().UnixNano())), g: g}, nil
}

func (ext imprvIKNPNCO) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// receive code words
	c := make([][]byte, ext.k)
	for row := range c {
		c[row] = make([]byte, ext.k)
		if _, err := io.ReadFull(rw, c[row]); err != nil {
			return err
		}
	}

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

	// receive masked columns u
	u := make([]byte, ext.m)
	q := make([][]byte, ext.k)
	for col := range q {
		if _, err = io.ReadFull(rw, u); err != nil {
			return err
		}

		q[col] = crypto.PseudorandomGeneratorWithBlake3(ext.g, seeds[col], ext.m)

		if s[col] == 0 {
			q[col], err = util.XorBytes(u, q[col])
			if err != nil {
				return err
			}
		}
	}

	q = util.Transpose(q)

	// encrypt messages and send them
	var key, ciphertext []byte
	for i := range messages {
		for choice, plaintext := range messages[i] {
			key, err = util.AndBytes(c[choice], s)
			if err != nil {
				return err
			}
			key, err = util.XorBytes(q[i], key)
			if err != nil {
				return err
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

func (ext imprvIKNPNCO) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	// sample random 16 byte secret key for AES-128
	sk := make([]uint8, 16)
	if _, err = ext.prng.Read(sk); err != nil {
		return nil
	}

	// compute code word using pseudorandom code on choice stirng r
	c := make([][]byte, ext.k)
	for row := range c {
		c[row] = crypto.PseudorandomCode(sk, ext.k, []byte{byte(row)})
		if _, err := rw.Write(c[row]); err != nil {
			return err
		}
	}

	d := make([][]byte, ext.m)
	for row := range d {
		d[row] = c[choices[row]]
	}

	d = util.Transpose(d)
	// sample k x k bit mtrix
	seeds, err := util.SampleRandomBitMatrix(ext.prng, 2*ext.k, ext.k)
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
	for col := range d {
		u = crypto.PseudorandomGeneratorWithBlake3(ext.g, baseMsgs[col][1], ext.m)

		t[col], _ = util.XorBytes(d[col], u)

		u = crypto.PseudorandomGeneratorWithBlake3(ext.g, baseMsgs[col][0], ext.m)

		u, _ = util.XorBytes(u, t[col])

		// send w
		if _, err = rw.Write(u); err != nil {
			return err
		}
	}

	t = util.Transpose(t)

	// receive encrypted messages.
	e := make([][]byte, ext.n)
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
