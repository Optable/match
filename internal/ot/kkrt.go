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
	"crypto/aes"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

type kkrt struct {
	baseOT OT
	m      int
	k      int
	n      int
	msgLen []int
}

func NewKKRT(m, k, n, baseOT int, ristretto bool, msgLen []int) (kkrt, error) {
	// send k columns of m bit messages
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = (m + util.PadTill512(m)) / 8
	}

	ot, err := NewBaseOT(baseOT, ristretto, k, crypto.P256, baseMsgLen, crypto.XORBlake3)
	if err != nil {
		return kkrt{}, err
	}

	return kkrt{baseOT: ot, m: m, k: k, n: n, msgLen: msgLen}, nil
}

func (ext kkrt) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]uint8, 16)
	if _, err = rand.Read(sk); err != nil {
		return nil
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

	// act as receiver in baseOT to receive q^j
	q := make([][]uint8, ext.k)
	if err = ext.baseOT.Receive(s, q, rw); err != nil {
		return err
	}

	// transpose q to m x k matrix for easier row operations
	q = util.TransposeByteMatrix(q)

	var key, ciphertext []byte
	aesBlock, _ := aes.NewCipher(sk)
	// encrypt messages and send them
	for i := range messages {
		// proof of concept, suppose we have n messages, and the choice string is an integer in [1, ..., n]
		for choice, plaintext := range messages[i] {
			// compute q_i ^ (C(r) & s)
			key = crypto.PseudorandomCode(aesBlock, []byte{byte(choice)})
			util.ConcurrentDoubleBitOp(util.AndXor, key, s, q[i])

			ciphertext, err = crypto.Encrypt(crypto.XORBlake3, key, uint8(choice), plaintext)
			if err != nil {
				return err
			}

			// write ciphertext to reciever
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

	// compute code word using pseudorandom code on choice stirng r in a separate thread
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

	// Sample k x m (padded column-wise to multiple of 8 uint64 (512 bits)) matrix T
	t, err := util.SampleRandomBitMatrix(rand.Reader, ext.k, ext.m)
	if err != nil {
		return err
	}
	fmt.Println("bor4")
	d := <-pseudorandomChan

	// make k pairs of m bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][][]byte, ext.k)
	for i := range baseMsgs {

		err = util.ConcurrentBitOp(util.Xor, d[i], t[i])
		if err != nil {
			return err
		}

		baseMsgs[i] = make([][]byte, 2)
		baseMsgs[i][0] = t[i]
		baseMsgs[i][1] = d[i]
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return err
	}

	t = util.TransposeByteMatrix(t)

	e := make([][]byte, ext.n)
	for i := range choices {
		// compute nb of bytes to be read
		l := crypto.EncryptLen(crypto.XORBlake3, ext.msgLen[i])
		// read all msg
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
