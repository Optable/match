package ot

import (
	"fmt"
	"io"

	gr "github.com/bwesterb/go-ristretto"
	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

/*
1 out of 2 base OT
from the paper: The Simplest Protocol for Oblivious Transfer
by Tung Chou and Claudio Orlandi in 2015
Reference: https://eprint.iacr.org/2015/267.pdf

Simplest OT but implemented with Ristretto points for the elliptic curve operation.
*/

type simplestRistretto struct {
	baseCount  int
	msgLen     []int
	cipherMode int
}

func newSimplestRistretto(baseCount int, msgLen []int, cipherMode int) (simplestRistretto, error) {
	if len(msgLen) != baseCount {
		return simplestRistretto{}, ErrBaseCountMissMatch
	}
	return simplestRistretto{baseCount: baseCount, msgLen: msgLen, cipherMode: cipherMode}, nil
}

func (s simplestRistretto) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	if len(messages) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	r := newRistrettoReader(rw)
	w := newRistrettoWriter(rw)

	// generate sender secret public key pairs
	secretA, pointA := generateKeys()
	// T = aA
	var pointT gr.Point
	pointT.ScalarMult(&pointA, &secretA)

	// send point A to receiver
	if err := w.write(&pointA); err != nil {
		return err
	}

	// make a slice of ristretto points to receive B from receiver.
	pointB := make([]gr.Point, s.baseCount)
	for i := range pointB {
		if err := r.read(&pointB[i]); err != nil {
			return err
		}
	}

	pointK := make([]gr.Point, 2)
	// encrypt plaintext messages and send it.
	for i := 0; i < s.baseCount; i++ {
		// k0 = aB
		pointK[0].ScalarMult(&pointB[i], &secretA)
		//k1 = a(B - A) = aB - aA
		pointK[1].Sub(&pointK[0], &pointT)

		// Encrypt plaintext message with key derived from received points B
		for choice, plaintext := range messages[i] {
			// derive key for encryption
			key, err := deriveKeyRistretto(&pointK[choice])
			if err != nil {
				return err
			}

			// encrypt
			ciphertext, err := crypto.Encrypt(s.cipherMode, key, uint8(choice), plaintext)
			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// send ciphertext
			if _, err = w.w.Write(ciphertext); err != nil {
				return err
			}
		}
	}

	return
}

func (s simplestRistretto) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices)*8 != len(messages) || len(choices)*8 != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	r := newRistrettoReader(rw)
	w := newRistrettoWriter(rw)

	// Receive point A from sender
	var pointA gr.Point
	if err := r.read(&pointA); err != nil {
		return err
	}

	// Generate points B, 1 for each OT,
	bSecrets := make([]gr.Scalar, s.baseCount)
	var pointB gr.Point
	for i := 0; i < s.baseCount; i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		bSecrets[i], pointB = generateKeys()

		// for each choice bit, compute the resultant point B and send it
		if util.TestBitSetInByte(choices, i) == 0 {
			if err := w.write(&pointB); err != nil {
				return err
			}
		} else {
			// B = A + bG
			pointB.Add(&pointA, &pointB)
			if err := w.write(&pointB); err != nil {
				return err
			}
		}
	}

	// receive encrypted messages, and decrypt it.
	e := make([][]byte, 2)
	var K gr.Point
	for i := 0; i < s.baseCount; i++ {
		// compute # of bytes to be read.
		l := crypto.EncryptLen(s.cipherMode, s.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err := io.ReadFull(r.r, e[j]); err != nil {
				return err
			}
		}

		// build keys for decryption
		K.ScalarMult(&pointA, &bSecrets[i])
		key, err := deriveKeyRistretto(&K)
		if err != nil {
			return err
		}

		// decrypt the message indexed by choice bit
		bit := util.TestBitSetInByte(choices, i)
		messages[i], err = crypto.Decrypt(s.cipherMode, key, bit, e[bit])
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
	}

	return
}
