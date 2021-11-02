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
from the paper: More Efficient Oblivious Transfer Extensions
by Gilad Asharov, Yehuda Lindell, Thomas Schneider, and Michael Zohner in 2017.
source: https://link.springer.com/content/pdf/10.1007/s00145-016-9236-6.pdf

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

func NewALSZ(m, k, baseOt, drbg int, ristretto bool, msgLen []int) (alsz, error) {
	// send k columns of messages of length k
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k / 8
	}

	ot, err := NewBaseOT(baseOt, ristretto, k, crypto.P256, baseMsgLen, crypto.XORBlake3)
	if err != nil {
		return alsz{}, err
	}

	return alsz{baseOT: ot, m: m, k: k, drbg: drbg, msgLen: msgLen}, nil
}

func (ext alsz) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// sample choice bits for baseOT
	s := make([]uint8, ext.k/8)
	if _, err = rand.Read(s); err != nil {
		return err
	}

	// act as receiver in baseOT to receive k x k bits seeds for the pseudorandom generator
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
		if util.BitSetInByte(s, col) {
			err = util.ConcurrentBitOp(util.Xor, q[col], u)
			if err != nil {
				return err
			}
		}
	}

	q = util.TransposeByteMatrix(q)

	// encrypt messages and send them
	var key, ciphertext []byte
	for i := range messages {
		for choice, plaintext := range messages[i] {
			key = q[i]
			if choice == 1 {
				err = util.ConcurrentBitOp(util.Xor, key, s)
				if err != nil {
					return err
				}
			}

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

func (ext alsz) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices)*8 != len(messages) || len(choices)*8 != ext.m {
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

	// pad choice to the right, the extra information will always end up in the bottom
	// once the matrix is transposed, and will never be used by both sender and receiver.
	paddedChoice := make([]byte, (ext.m+util.PadTill512(ext.m))/8)
	copy(paddedChoice, choices)

	var t = make([][]byte, ext.k)
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	var u = make([]byte, paddedLen)
	// u^i = G(seeds[1])
	// t^i = d^i ^ u^i
	for col := range t {
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

		err = util.ConcurrentBitOp(util.Xor, u, paddedChoice)
		if err != nil {
			return err
		}

		// send u^col
		if _, err = rw.Write(u); err != nil {
			return err
		}
	}

	// transpose t to m x k for easier row operations
	t = util.TransposeByteMatrix(t)

	// receive encrypted messages.
	e := make([][]byte, 2)
	for i := 0; i < ext.m; i++ {
		var choiceBit byte
		if util.BitSetInByte(choices, i) {
			choiceBit = 1
		}
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
		messages[i], err = crypto.Decrypt(crypto.XORBlake3, t[i], choiceBit, e[choiceBit])
		if err != nil {
			return fmt.Errorf("error decrypting sender messages: %s", err)
		}
	}

	return
}
