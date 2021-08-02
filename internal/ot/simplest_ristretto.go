package ot

import (
	"fmt"
	"io"

	gr "github.com/bwesterb/go-ristretto"
)

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

func (s simplestRistretto) Send(messages [][2][]byte, rw io.ReadWriter) (err error) {
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
			ciphertext, err := encrypt(s.cipherMode, key, uint8(choice), plaintext)
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
	if len(choices) != len(messages) || len(choices) != s.baseCount {
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
		switch choices[i] {
		case 0:
			if err := w.write(&B); err != nil {
				return err
			}
		case 1:
			// B = A + bG
			B.Add(&A, &B)
			if err := w.write(&B); err != nil {
				return err
			}
		default:
			return fmt.Errorf("choice bits should be binary, got %v", choices[i])
		}
	}

	// receive encrypted messages, and decrypt it.
	e := make([][]byte, 2)
	var K gr.Point
	for i := 0; i < s.baseCount; i++ {
		// compute # of bytes to be read.
		l := encryptLen(s.cipherMode, s.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err := io.ReadFull(r.r, e[j]); err != nil {
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
		messages[i], err = decrypt(s.cipherMode, key, choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
	}

	return
}
