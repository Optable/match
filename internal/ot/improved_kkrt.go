package ot

import (
	"crypto/aes"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

type imprvKKRT struct {
	baseOT OT
	m      int
	k      int
	n      int
	msgLen []int
	drbg   int
}

func NewImprovedKKRT(m, k, n, baseOt, drbg int, ristretto bool, msgLen []int) (imprvKKRT, error) {
	// send k columns of messages of length k
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k / 8
	}

	ot, err := NewBaseOT(baseOt, ristretto, k, crypto.P256, baseMsgLen, crypto.XORBlake3)
	if err != nil {
		return imprvKKRT{}, err
	}

	return imprvKKRT{baseOT: ot, m: m, k: k, n: n, msgLen: msgLen, drbg: drbg}, nil
}

func (ext imprvKKRT) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]uint8, 16)
	if _, err = rand.Read(sk); err != nil {
		return err
	}

	// send the secret key
	if _, err := rw.Write(sk); err != nil {
		return err
	}

	// sample choice bits for baseOT
	s := make([]uint8, ext.k/8)
	if _, err = rand.Read(s); err != nil {
		return err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]uint8, ext.k)
	if err = ext.baseOT.Receive(s, seeds, rw); err != nil {
		return err
	}

	// receive masked columns u
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	u := make([]byte, paddedLen)
	q := make([][]byte, ext.k)
	for col := range q {
		if _, err = io.ReadFull(rw, u); err != nil {
			return err
		}

		q[col], err = crypto.PseudorandomGenerate(ext.drbg, seeds[col], paddedLen)
		if err != nil {
			return err
		}

		// Binary AND of each byte in u with the test bit
		// if bit is 1, we get whole row u to XOR with q[col]
		// if bit is 0, we get a row of 0s which when XORed
		// with q[col] just returns the same row so no need to do
		// an operation
		if util.TestBitSetInByte(s, col) == 1 {
			err = util.ConcurrentBitOp(util.Xor, q[col], u)
			if err != nil {
				return err
			}
		}
	}

	q = util.TransposeByteMatrix(q)

	// encrypt messages and send them
	var key, ciphertext []byte
	aesBlock, _ := aes.NewCipher(sk)
	for i := range messages {
		for choice, plaintext := range messages[i] {
			// compute q_i ^ (C(r) & s)
			key = crypto.PseudorandomCode(aesBlock, []byte{byte(choice)})
			util.ConcurrentDoubleBitOp(util.AndXor, key, s, q[i])

			ciphertext, err = crypto.Encrypt(crypto.XORBlake3, key, uint8(choice), plaintext)
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
	var pseudorandomChan = make(chan [][]byte)
	go func() {
		d := make([][]byte, ext.m)
		aesBlock, _ := aes.NewCipher(sk)
		for i := 0; i < ext.m; i++ {
			d[i] = crypto.PseudorandomCode(aesBlock, []byte{choices[i]})
		}
		tr := util.TransposeByteMatrix(d)
		pseudorandomChan <- tr
	}()

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

	d := <-pseudorandomChan
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	var t = make([][]byte, ext.k)
	var u = make([]byte, paddedLen)
	// u^i = G(seeds[1])
	// t^i = d^i ^ u^i
	for col := range d {
		t[col], err = crypto.PseudorandomGenerate(ext.drbg, baseMsgs[col][0], paddedLen)
		if err != nil {
			return err
		}

		u, err = crypto.PseudorandomGenerate(ext.drbg, baseMsgs[col][1], paddedLen)
		if err != nil {
			return err
		}
		err = util.ConcurrentBitOp(util.Xor, u, t[col])
		if err != nil {
			return err
		}
		err = util.ConcurrentBitOp(util.Xor, u, d[col])
		if err != nil {
			return err
		}

		// send w
		if _, err = rw.Write(u); err != nil {
			return err
		}
	}

	t = util.TransposeByteMatrix(t)

	// receive encrypted messages.
	e := make([][]byte, ext.n)
	for i := range choices {
		// compute nb of bytes to be read
		l := crypto.EncryptLen(crypto.XORBlake3, ext.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// decrypt received ciphertext using key (choices[i], t_i)
		messages[i], err = crypto.Decrypt(crypto.XORBlake3, t[i], choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("error decrypting sender messages: %s", err)
		}
	}

	return
}
