package ot

import (
	"crypto/aes"
	"crypto/elliptic"
	"crypto/rand"
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

	for i, _ := range messages {
		if len(messages[i][0]) != len(messages[i][1]) {
			return fmt.Errorf("Expecting the length of the pair of messages to be the same, got %d, %d\n", len(messages[i][0]), len(messages[i][1]))
		}
	}

	// Instantiate Reader, Writer
	r := newReader(rw, s.curve, s.encodeLen)
	w := newWriter(rw, s.curve)

	// generate sender secret public key pairs
	a, Ax, Ay, err := elliptic.GenerateKey(s.curve, rand.Reader)
	if err != nil {
		return err
	}

	// send point A in marshaled []byte to receiver
	if err := w.write(newPoints(Ax, Ay)); err != nil {
		return err
	}

	// make a slice of point B, 1 for each OT, and receive them
	B := make([]points, s.baseCount)
	for i, _ := range B {
		B[i] = newPoints(new(big.Int), new(big.Int))
		if err := r.read(B[i]); err != nil {
			return err
		}
	}

	// A = aA
	Ax, Ay = s.curve.ScalarMult(Ax, Ay, a)
	Ay.Neg(Ay) // -Ay
	var kx, ky *big.Int
	var k, ciphertext []byte

	// encrypt plaintext messages and send it.
	for i := 0; i < s.baseCount; i++ {
		// sanity check
		if !s.curve.IsOnCurve(B[i].x, B[i].y) {
			return fmt.Errorf("Point A received from sender is not on curve: %s", s.curve.Params().Name)
		}

		// Encrypt plaintext message with key derived from received points B
		for b, plaintext := range messages[i] {
			// precompute k0 = aB
			kx, ky = s.curve.ScalarMult(B[i].x, B[i].y, a)
			if b == 1 {
				//k1 = a(B - A) = aB - aA
				kx, ky = s.curve.Add(kx, ky, Ax, Ay)
			}

			// derive key for aes
			k = deriveKey(elliptic.Marshal(s.curve, kx, ky))

			// instantiate AES
			block, err := aes.NewCipher(k)
			if err != nil {
				return err
			}

			// encrypt plaintext using aes GCM mode
			ciphertext, err = encrypt(block, plaintext)
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

func (s simplest) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	r := newReader(rw, s.curve, s.encodeLen)
	w := newWriter(rw, s.curve)

	// Receive marshalled point A from sender
	A := newPoints(new(big.Int), new(big.Int))
	if err := r.read(A); err != nil {
		return err
	}

	// sanity check
	if !s.curve.IsOnCurve(A.x, A.y) {
		return fmt.Errorf("Point A received from sender is not on curve: %s", s.curve.Params().Name)
	}

	// Generate points B, 1 for each OT
	bSecrets := make([][]byte, s.baseCount)
	var Bx, By *big.Int
	var b []byte
	for i := 0; i < s.baseCount; i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		b, Bx, By, err = elliptic.GenerateKey(s.curve, rand.Reader)
		if err != nil {
			return err
		}
		bSecrets[i] = b

		// for each choice bit, compute the resultant point B and send it
		switch choices[i] {
		case 0:
			if err := w.write(newPoints(Bx, By)); err != nil {
				return err
			}
		case 1:
			// B = A + bG
			Bx, By = s.curve.Add(A.x, A.y, Bx, By)
			if err := w.write(newPoints(Bx, By)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Choice bits should be binary, got %v", choices[i])
		}
	}

	// receive encrypted messages, and decrypt it.
	e := make([][]byte, 2)
	var kx, ky *big.Int
	var k []byte

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
		kx, ky = s.curve.ScalarMult(A.x, A.y, bSecrets[i])
		k = deriveKey(elliptic.Marshal(s.curve, kx, ky))
		// instantiate AES
		block, err := aes.NewCipher(k)
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
