package ot

import (
	"crypto/aes"
	"fmt"
	"io"

	gr "github.com/bwesterb/go-ristretto"
)

type simplestRistretto struct {
	baseCount int
	msgLen    []int
}

func newSimplestRistretto(baseCount int, msgLen []int) (simplestRistretto, error) {
	if len(msgLen) != baseCount {
		return simplestRistretto{}, ErrBaseCountMissMatch
	}
	return simplestRistretto{baseCount: baseCount, msgLen: msgLen}, nil
}

func (s simplestRistretto) Send(messages [][2][]byte, rw io.ReadWriter) (err error) {
	if len(messages) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	for i, _ := range messages {
		if len(messages[i][0]) != len(messages[i][1]) {
			return fmt.Errorf("Expecting the length of the pair of messages to be the same, got %d, %d\n", len(messages[i][0]), len(messages[i][1]))
		}
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

	// make a slice of ristretto point B, 1 for each OT, and receive them
	B := make([]gr.Point, s.baseCount)
	for i, _ := range B {
		if err := r.read(&B[i]); err != nil {
			return err
		}
	}

	var K gr.Point
	key := make([]byte, encodeLen)
	// encrypt plaintext messages and send it.
	for i := 0; i < s.baseCount; i++ {
		// Encrypt plaintext message with key derived from received points B
		for choice, plaintext := range messages[i] {
			// precompute k0 = aB
			K.ScalarMult(&B[i], &a)
			if choice == 1 {
				//k1 = a(B - A) = aB - aA
				K.Sub(&K, &T)
			}

			// derive key for aes
			key, err = deriveKeyRistretto(&K)
			if err != nil {
				return err
			}

			// instantiate AES
			block, err := aes.NewCipher(key)
			if err != nil {
				return err
			}

			// encrypt plaintext using aes GCM mode
			ciphertext, err := encrypt(block, plaintext)
			if err != nil {
				return fmt.Errorf("Error encrypting sender message: %s\n", err)
			}

			// send ciphertext
			if _, err = w.w.Write(ciphertext); err != nil {
				return err
			}
		}
	}

	return nil
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
			return fmt.Errorf("Choice bits should be binary, got %v", choices[i])
		}
	}

	// receive encrypted messages, and decrypt it.
	e := make([][]byte, 2)
	var K gr.Point
	key := make([]byte, encodeLen)
	for i := 0; i < s.baseCount; i++ {
		// compute # of bytes to be read.
		l := encryptLen(s.msgLen[i])
		// read both msg
		for j, _ := range e {
			e[j] = make([]byte, l)
			if _, err := io.ReadFull(r.r, e[j]); err != nil {
				return err
			}
		}

		// build keys for decrypting choice messages
		K.ScalarMult(&A, &bSecrets[i])
		key, err = deriveKeyRistretto(&K)
		// instantiate AES
		block, err := aes.NewCipher(key)
		if err != nil {
			return err
		}

		// decrypt the message indexed by choice bit
		messages[i], err = decrypt(block, e[choices[i]])
		if err != nil {
			return fmt.Errorf("Error encrypting sender message: %s\n", err)
		}
	}

	return nil
}
