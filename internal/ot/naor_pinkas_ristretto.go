package ot

import (
	"fmt"
	"io"

	gr "github.com/bwesterb/go-ristretto"
)

type naorPinkasRistretto struct {
	baseCount  int
	msgLen     []int
	cipherMode int
}

func newNaorPinkasRistretto(baseCount int, msgLen []int, cipherMode int) (naorPinkasRistretto, error) {
	if len(msgLen) != baseCount {
		return naorPinkasRistretto{}, ErrBaseCountMissMatch
	}
	return naorPinkasRistretto{baseCount: baseCount, msgLen: msgLen, cipherMode: cipherMode}, nil
}

func (n naorPinkasRistretto) Send(messages [][2][]byte, rw io.ReadWriter) (err error) {
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
			ciphertext, err := encrypt(n.cipherMode, key, uint8(choice), plaintext)
			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// send ciphertext
			if _, err = writer.w.Write(ciphertext); err != nil {
				return err
			}
		}
	}

	return
}

func (n naorPinkasRistretto) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != n.baseCount {
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
		switch choices[i] {
		case 0:
			// K0 = Kc = B
			// K1 = K1-c = A - B
			if err := writer.write(&B); err != nil {
				return err
			}
		case 1:
			// K1 = Kc = B
			// K0 = K1-c = A - B
			B.Sub(&A, &B)
			if err := writer.write(&B); err != nil {
				return err
			}
		default:
			return fmt.Errorf("choice bits should be binary, got %v", choices[i])
		}
	}

	e := make([][]byte, 2)
	var K gr.Point
	// receive encrypted messages, and decrypt it.
	for i := 0; i < n.baseCount; i++ {
		// compute # of bytes to be read.
		l := encryptLen(n.cipherMode, n.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err := io.ReadFull(reader.r, e[j]); err != nil {
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
		messages[i], err = decrypt(n.cipherMode, key, choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
	}

	return
}
