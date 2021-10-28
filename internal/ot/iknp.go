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
*/

type iknp struct {
	baseOT OT
	m      int
	k      int
	msgLen []int
}

func NewIKNP(m, k, baseOT int, ristretto bool, msgLen []int) (iknp, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = (m + util.PadTill512(m)) / 8
	}

	ot, err := NewBaseOT(baseOT, ristretto, k, crypto.P256, baseMsgLen, crypto.XORBlake3)
	if err != nil {
		return iknp{}, err
	}

	return iknp{baseOT: ot, m: m, k: k, msgLen: msgLen}, nil
}

func (ext iknp) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
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

func (ext iknp) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices)*8 != len(messages) || len(messages) != ext.m {
		return ErrBaseCountMissMatch
	}

	// Sample k x m matrix T
	t, err := util.SampleRandomBitMatrix(rand.Reader, ext.k, ext.m)
	if err != nil {
		return err
	}

	// pad choice to the right, the extra information will always end up in the bottom
	// once the matrix is transposed, and will never be used by both sender and receiver.
	paddedChoice := make([]byte, (ext.m+util.PadTill512(ext.m))/8)
	copy(paddedChoice, choices)

	// make k pairs of m bytes baseOT messages: {t^j, t^j xor choices}
	baseMsgs := make([][][]byte, ext.k)
	for i := range baseMsgs {
		baseMsgs[i] = make([][]byte, 2)
		baseMsgs[i][0] = t[i]
		baseMsgs[i][1], err = util.XorBytes(t[i], paddedChoice)
		if err != nil {
			return err
		}
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return err
	}

	// compute m x k transpose to access columns easier
	t = util.TransposeByteMatrix(t)

	e := make([][]byte, 2)
	for i := 0; i < ext.m; i++ {
		var choiceBit byte
		if util.BitSetInByte(choices, i) {
			choiceBit = 1
		}
		// compute # of bytes to be read
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
