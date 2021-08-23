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
from the paper: Efficient Oblivious Transfer Protocol
by Moni Naor and Benny Pinkas in 2001.
reference: https://dl.acm.org/doi/abs/10.5555/365411.365502

Naor-Pinkas OT implemented using Ristretto points for the elliptic curve operations.
*/

type naorPinkasRistrettoBitSet struct {
	baseCount  int
	msgLen     []int
	cipherMode int
}

func newNaorPinkasRistrettoBitSet(baseCount int, msgLen []int, cipherMode int) (naorPinkasRistrettoBitSet, error) {
	if len(msgLen) != baseCount {
		return naorPinkasRistrettoBitSet{}, ErrBaseCountMissMatch
	}
	return naorPinkasRistrettoBitSet{baseCount: baseCount, msgLen: msgLen, cipherMode: cipherMode}, nil
}

func (n naorPinkasRistrettoBitSet) Send(messages [][]*bitset.BitSet, rw io.ReadWriter) (err error) {
	if len(messages) != n.baseCount {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	reader := newReaderRistretto(rw)
	writer := newWriterRistretto(rw)

	// generate sender A point w/o secret, since a is never used.
	var A gr.Point
	A.Rand()
	// generate sender secret public key pairs used for encryption
	r, R := generateKeys()

	// send both public keys to receiver
	if err := writer.write(&A); err != nil {
		return err
	}
	if err := writer.write(&R); err != nil {
		return err
	}

	// precompute A = rA
	A.ScalarMult(&A, &r)

	// make a slice of ristretto points to receive K0.
	pointK0 := make([]gr.Point, n.baseCount)
	for i := range pointK0 {
		if err := reader.read(&pointK0[i]); err != nil {
			return err
		}
	}

	K := make([]gr.Point, 2)
	// encrypt plaintext message and send them.
	for i := 0; i < n.baseCount; i++ {
		// compute K0 = rK0
		K[0].ScalarMult(&pointK0[i], &r)
		// compute K1 = rA - rK0
		K[1].Sub(&A, &K[0])

		// encrypt plaintext message with key derived from K0, K1
		for choice, plaintext := range messages[i] {
			// derive key for encryption
			key, err := deriveKeyRistretto(&K[choice])
			if err != nil {
				return err
			}

			// encrypt
			// ciphertext, err := cipher.Encrypt(n.cipherMode, key, uint8(choice), util.BitSetToBits(plaintext))
			ciphertext, err := cipher.Encrypt(n.cipherMode, key, uint8(choice), util.BitSetToBytes(plaintext))
			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// convert ciphertext to BitSet
			cipherBitSet := util.BytesToBitSet(ciphertext)

			// send ciphertext
			if _, err = cipherBitSet.WriteTo(rw); err != nil {
				return err
			}
		}
	}

	return
}

func (n naorPinkasRistrettoBitSet) Receive(choices *bitset.BitSet, messages []*bitset.BitSet, rw io.ReadWriter) (err error) {
	if int(choices.Len()) < len(messages) || int(choices.Len()) > len(messages)+63 || len(messages) != n.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	reader := newReaderRistretto(rw)
	writer := newWriterRistretto(rw)

	// Receive point A from sender
	var A gr.Point
	if err := reader.read(&A); err != nil {
		return err
	}

	// Receive point R from sender
	var R gr.Point
	if err := reader.read(&R); err != nil {
		return err
	}

	// Generate points B, 1 for each OT,
	bSecrets := make([]gr.Scalar, n.baseCount)
	for i := 0; i < n.baseCount; i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		b, B := generateKeys()
		if err != nil {
			return err
		}
		bSecrets[i] = b

		// for each choice bit, compute the resultant point Kc, K1-c and send K0
		if choices.Test(uint(i)) {
			// K1 = Kc = B
			// K0 = K1-c = A - B
			B.Sub(&A, &B)
			if err := writer.write(&B); err != nil {
				return err
			}
		} else {
			// K0 = Kc = B
			// K1 = K1-c = A - B
			if err := writer.write(&B); err != nil {
				return err
			}
		}
	}

	e := make([]*bitset.BitSet, 2)
	var K gr.Point
	// receive encrypted messages, and decrypt it.
	for i := 0; i < n.baseCount; i++ {
		// compute # of bytes to be read.
		l := cipher.EncryptLen(n.cipherMode, n.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = bitset.New(uint(l))
			if _, err := e[j].ReadFrom(rw); err != nil {
				return err
			}
		}

		// build keys for decrypting choice messages
		// K = bR
		K.ScalarMult(&R, &bSecrets[i])
		key, err := deriveKeyRistretto(&K)
		if err != nil {
			return err
		}

		// decrypt the message indexed by choice bit
		var choice uint8
		if choices.Test(uint(i)) {
			choice = 1
		}
		// message, err := cipher.Decrypt(n.cipherMode, key, choice, util.BitSetToBits(e[choice]))
		message, err := cipher.Decrypt(n.cipherMode, key, choice, util.BitSetToBytes(e[choice]))
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
		messages[i] = util.BytesToBitSet(message)
	}

	return
}
