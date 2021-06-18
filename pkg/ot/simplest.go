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

func (s simplest) Send(messages [][2][]byte, rw io.ReadWriter) error {
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
	// build keys for encypting messages
	for i := 0; i < s.baseCount; i++ {
		// sanity check
		if !s.curve.IsOnCurve(B[i].x, B[i].y) {
			return fmt.Errorf("Point A received from sender is not on curve: %s", s.curve.Params().Name)
		}

		// k0 = aB
		k0x, k0y := s.curve.ScalarMult(B[i].x, B[i].y, a)
		// encrypt message[0]
		k0 := deriveKey(elliptic.Marshal(s.curve, k0x, k0y))
		// instantiate AES
		block1, err := aes.NewCipher(k0)
		if err != nil {
			return err
		}

		m0, err := encrypt(block1, messages[i][0])
		if err != nil {
			return fmt.Errorf("Error encrypting sender message: %s\n", err)
		}

		// send encrypted m0
		if _, err := w.w.Write(m0); err != nil {
			return err
		}

		//k1 = a(B - A) = aB - aA
		k1x, k1y := s.curve.Add(k0x, k0y, Ax, Ay)
		// encrypt message[1]
		k1 := deriveKey(elliptic.Marshal(s.curve, k1x, k1y))
		// instantiate AES
		block2, err := aes.NewCipher(k1)
		if err != nil {
			return err
		}

		m1, err := encrypt(block2, messages[i][1])
		if err != nil {
			return fmt.Errorf("Error encrypting sender message: %s\n", err)
		}

		// send encrypted m1
		if _, err := w.w.Write(m1); err != nil {
			return err
		}
	}

	return nil
}

func (s simplest) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error {
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
	for i := 0; i < s.baseCount; i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		b, Bx, By, err := elliptic.GenerateKey(s.curve, rand.Reader)
		if err != nil {
			return err
		}
		bSecrets[i] = b

		// for each choice bit, compute the resultant point B
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

	// receive encrypted messages
	enc := make([][]byte, s.baseCount)
	for i, _ := range enc {
		// read both msg
		e := make([][]byte, 2)
		l := encryptLen(s.msgLen[i])
		for j, _ := range e {
			e[j] = make([]byte, l)
			if _, err := io.ReadFull(r.r, e[j]); err != nil {
				return err
			}
		}
		// save the chosen msg
		enc[i] = e[choices[i]]
	}

	// build keys for encypting messages
	for i := 0; i < s.baseCount; i++ {
		// right decryption key
		kx, ky := s.curve.ScalarMult(A.x, A.y, bSecrets[i])
		k := deriveKey(elliptic.Marshal(s.curve, kx, ky))
		// instantiate AES
		block, err := aes.NewCipher(k)
		if err != nil {
			return err
		}

		m, err := decrypt(block, enc[i])
		if err != nil {
			return fmt.Errorf("Error encrypting sender message: %s\n", err)
		}

		// copy decrypted message to messages
		messages[i] = make([]byte, len(m))
		copy(messages[i], m)
	}

	return nil
}
