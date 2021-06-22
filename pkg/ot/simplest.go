package ot

import (
	"crypto/aes"
	"crypto/elliptic"
	"fmt"
	"io"
	"math/big"
)

type simplest struct {
	baseCount int
	curve     elliptic.Curve
	encodeLen int
	msgLen    []int
}

func newSimplest(baseCount int, curveName string, msgLen []int) (simplest, error) {
	if len(msgLen) != baseCount {
		return simplest{}, ErrBaseCountMissMatch
	}
	curve, encodeLen := initCurve(curveName)
	return simplest{baseCount: baseCount, curve: curve, encodeLen: encodeLen, msgLen: msgLen}, nil
}

func (s simplest) Send(messages [][2][]byte, rw io.ReadWriter) (err error) {
	if len(messages) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	reader := newReader(rw, s.curve, s.encodeLen)
	writer := newWriter(rw, s.curve)

	// generate sender secret public key pairs
	a, A, err := generateKeyWithPoints(s.curve)
	if err != nil {
		return err
	}

	// send point A in marshaled []byte to receiver
	if err := writer.write(A); err != nil {
		return err
	}

	// Precompute A = aA
	A = A.scalarMult(a)

	// make a slice of point B, 1 for each OT, and receive them
	B := make([]points, s.baseCount)
	for i, _ := range B {
		B[i] = newPoints(s.curve, new(big.Int), new(big.Int))
		if err := reader.read(B[i]); err != nil {
			return err
		}
	}

	K := make([]points, 2)
	var ciphertext []byte
	// encrypt plaintext messages and send it.
	for i := 0; i < s.baseCount; i++ {
		// sanity check
		if !B[i].isOnCurve() {
			return fmt.Errorf("Point A received from sender is not on curve: %s", s.curve.Params().Name)
		}

		// k0 = aB
		K[0] = B[i].scalarMult(a)
		//k1 = a(B - A) = aB - aA
		K[1] = K[0].sub(A)

		// Encrypt plaintext message with key derived from received points B
		for choice, plaintext := range messages[i] {
			// derive key and instantiate AES
			block, err := aes.NewCipher(K[choice].deriveKey())
			if err != nil {
				return err
			}

			// encrypt plaintext using aes GCM mode
			ciphertext, err = encrypt(block, plaintext)
			if err != nil {
				return fmt.Errorf("Error encrypting sender message: %s\n", err)
			}

			// send ciphertext
			if _, err = writer.w.Write(ciphertext); err != nil {
				return err
			}
		}
	}

	return
}

func (s simplest) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	reader := newReader(rw, s.curve, s.encodeLen)
	writer := newWriter(rw, s.curve)

	// Receive marshalled point A from sender
	A := newPoints(s.curve, new(big.Int), new(big.Int))
	if err := reader.read(A); err != nil {
		return err
	}

	// sanity check
	if !A.isOnCurve() {
		return fmt.Errorf("Point A received from sender is not on curve: %s", s.curve.Params().Name)
	}

	// Generate points B, 1 for each OT
	bSecrets := make([][]byte, s.baseCount)
	var B points
	var b []byte
	for i := 0; i < s.baseCount; i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		b, B, err = generateKeyWithPoints(s.curve)
		if err != nil {
			return err
		}
		bSecrets[i] = b

		// for each choice bit, compute the resultant point B and send it
		switch choices[i] {
		case 0:
			// B
			if err := writer.write(B); err != nil {
				return err
			}
		case 1:
			// B = A + B
			if err := writer.write(A.add(B)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Choice bits should be binary, got %v", choices[i])
		}
	}

	e := make([][]byte, 2)
	var K points
	// receive encrypted messages, and decrypt it.
	for i := 0; i < s.baseCount; i++ {
		// compute # of bytes to be read.
		l := encryptLen(s.msgLen[i])
		// read both msg
		for j, _ := range e {
			e[j] = make([]byte, l)
			if _, err := io.ReadFull(reader.r, e[j]); err != nil {
				return err
			}
		}

		// build keys for decrypting choice messages
		K = A.scalarMult(bSecrets[i])
		// instantiate AES
		block, err := aes.NewCipher(K.deriveKey())
		if err != nil {
			return err
		}

		// decrypt the message indexed by choice bit
		messages[i], err = decrypt(block, e[choices[i]])
		if err != nil {
			return fmt.Errorf("Error encrypting sender message: %s\n", err)
		}
	}

	return
}
