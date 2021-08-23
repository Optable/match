package ot

import (
	"fmt"
	"io"

	"github.com/bits-and-blooms/bitset"
	gr "github.com/bwesterb/go-ristretto"
	"github.com/optable/match/internal/cipher"
	"github.com/optable/match/internal/util"
)

/*
1 out of 2 base OT
from the paper: The Simplest Protocol for Oblivious Transfer
by Tung Chou and Claudio Orlandi in 2015
Reference: https://eprint.iacr.org/2015/267.pdf

Simplest OT but implemented with Ristretto points for the elliptic curve operation.
*/

type simplestRistrettoBitSet struct {
	baseCount  int
	msgLen     []int
	cipherMode int
}

func newSimplestRistrettoBitSet(baseCount int, msgLen []int, cipherMode int) (simplestRistrettoBitSet, error) {
	if len(msgLen) != baseCount {
		return simplestRistrettoBitSet{}, ErrBaseCountMissMatch
	}
	return simplestRistrettoBitSet{baseCount: baseCount, msgLen: msgLen, cipherMode: cipherMode}, nil
}

func (s simplestRistrettoBitSet) Send(messages [][]*bitset.BitSet, rw io.ReadWriter) (err error) {
	if len(messages) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	r := newReaderRistretto(rw)
	w := newWriterRistretto(rw)

	// generate sender secret public key pairs
	a, A := generateKeys()
	// T = aA
	var T gr.Point
	T.ScalarMult(&A, &a)

	// send point A to receiver
	if err := w.write(&A); err != nil {
		return err
	}

	// make a slice of ristretto points to receive B from receiver.
	B := make([]gr.Point, s.baseCount)
	for i := range B {
		if err := r.read(&B[i]); err != nil {
			return err
		}
	}

	K := make([]gr.Point, 2)
	// encrypt plaintext messages and send it.
	for i := 0; i < s.baseCount; i++ {
		// k0 = aB
		K[0].ScalarMult(&B[i], &a)
		//k1 = a(B - A) = aB - aA
		K[1].Sub(&K[0], &T)

		// Encrypt plaintext message with key derived from received points B
		for choice, plaintext := range messages[i] {
			// derive key for encryption
			key, err := deriveKeyRistretto(&K[choice])
			if err != nil {
				return err
			}

			// encrypt
			// ciphertext, err := cipher.Encrypt(s.cipherMode, key, uint8(choice), util.BitSetToBits(plaintext))
			ciphertext, err := cipher.Encrypt(s.cipherMode, key, uint8(choice), util.BitSetToBytes(plaintext))
			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// convert ciphertext into BitSet
			cipherBitSet := util.BytesToBitSet(ciphertext)

			// send ciphertext
			if _, err = cipherBitSet.WriteTo(rw); err != nil {
				return err
			}
		}
	}

	return
}

func (s simplestRistrettoBitSet) Receive(choices *bitset.BitSet, messages []*bitset.BitSet, rw io.ReadWriter) (err error) {
	if int(choices.Len()) < len(messages) || int(choices.Len()) > len(messages)+63 || len(messages) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	r := newReaderRistretto(rw)
	w := newWriterRistretto(rw)

	// Receive point A from sender
	var A gr.Point
	if err := r.read(&A); err != nil {
		return err
	}

	// Generate points B, 1 for each OT,
	bSecrets := make([]gr.Scalar, s.baseCount)
	for i := 0; i < s.baseCount; i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		b, B := generateKeys()
		if err != nil {
			return err
		}
		bSecrets[i] = b

		// for each choice bit, compute the resultant point B and send it
		if choices.Test(uint(i)) {
			// B = A + bG
			B.Add(&A, &B)
			if err := w.write(&B); err != nil {
				return err
			}
		} else {
			if err := w.write(&B); err != nil {
				return err
			}
		}
	}

	// receive encrypted messages, and decrypt it.
	e := make([]*bitset.BitSet, 2)
	var K gr.Point
	for i := 0; i < s.baseCount; i++ {
		// compute # of bytes to be read.
		l := cipher.EncryptLen(s.cipherMode, s.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = bitset.New(uint(l))
			if _, err := e[j].ReadFrom(rw); err != nil {
				return err
			}
		}

		// build keys for decryption
		K.ScalarMult(&A, &bSecrets[i])
		key, err := deriveKeyRistretto(&K)
		if err != nil {
			return err
		}

		// decrypt the message indexed by choice bit
		var choice uint8
		if choices.Test(uint(i)) {
			choice = 1
		}
		// message, err := cipher.Decrypt(s.cipherMode, key, choice, util.BitSetToBits(e[choice]))
		message, err := cipher.Decrypt(s.cipherMode, key, choice, util.BitSetToBytes(e[choice]))
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
		messages[i] = util.BytesToBitSet(message)
	}

	return
}
