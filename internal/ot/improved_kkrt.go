package ot

import (
	"crypto/aes"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

/*
1 out of N improved KKRT OT extension done by Justin Li
*/

type imprvKKRT struct {
	baseOT OT
	m      int
	k      int
	n      int
	msgLen []int
	prng   *rand.Rand
	g      *blake3.Hasher
}

func NewImprovedKKRT(m, k, n, baseOt int, ristretto bool, msgLen []int) (imprvKKRT, error) {
	// send k columns of messages of length k
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k
	}

	ot, err := NewBaseOT(baseOt, ristretto, k, iknpCurve, baseMsgLen, iknpCipherMode)
	if err != nil {
		return imprvKKRT{}, err
	}
	g := blake3.New()

	return imprvKKRT{baseOT: ot, m: m, k: k, n: n, msgLen: msgLen,
		prng: rand.New(rand.NewSource(time.Now().UnixNano())), g: g}, nil
}

func (ext imprvKKRT) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]uint8, 16)
	if _, err = ext.prng.Read(sk); err != nil {
		return err
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

		q[col], _ = util.XorBytes(util.AndByte(s[col], u), q[col])
	}

	q = util.Transpose(q)

	// encrypt messages and send them
	var key, ciphertext, x []byte
	aesBlock, _ := aes.NewCipher(sk)
	for i := range messages {
		for choice, plaintext := range messages[i] {
			// compute q_i ^ (C(r) & s)
			x, _ = util.AndBytes(crypto.PseudorandomCode(aesBlock, []byte{byte(choice)}), s)
			key, _ = util.XorBytes(q[i], x)

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

func (ext imprvKKRT) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	sk := make([]byte, 16)
	if _, err = io.ReadFull(rw, sk); err != nil {
		return err
	}

	// compute code word using pseudorandom code on choice stirng r
	d := make([][]byte, ext.m)
	aesBlock, _ := aes.NewCipher(sk)
	for row := range d {
		d[row] = crypto.PseudorandomCode(aesBlock, []byte{choices[row]})
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
		t[col] = crypto.PseudorandomGeneratorWithBlake3(ext.g, baseMsgs[col][0], ext.m)

		u = crypto.PseudorandomGeneratorWithBlake3(ext.g, baseMsgs[col][1], ext.m)
		u, _ = util.XorBytes(t[col], u)
		u, _ = util.XorBytes(u, d[col])

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
