package ot

import (
	"crypto/aes"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
)

type simplest struct {
	baseCount int
	curve     elliptic.Curve
}

func NewSimplest(baseCount int, curveName string) (*simplest, error) {
	return &simplest{baseCount: baseCount, curve: InitCurve(curveName)}, nil
}

func (s *simplest) Send(messages [][2][]byte, c chan []byte) error {
	if len(messages) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// generate sender secret public key pairs
	a, Ax, Ay, err := elliptic.GenerateKey(s.curve, rand.Reader)
	if err != nil {
		return err
	}

	// send point A in marshaled []byte to receiver
	A := elliptic.Marshal(s.curve, Ax, Ay)
	c <- A

	// make a slice of point B, 1 for each OT, and receive them
	var B [][]byte
	for b := range c {
		B = append(B, b)
	}

	//sanity check
	if len(B) != s.baseCount {
		return fmt.Errorf("Miss match with # of elements in channel and baseCount, got %d elements", len(B))
	}

	// A = aA
	Ax, Ay = s.curve.ScalarMult(Ax, Ay, a)
	Ay.Neg(Ay) // -Ay
	// build keys for encypting messages
	for i := 0; i < s.baseCount; i++ {
		// unmarshal point B
		Bx, By := elliptic.Unmarshal(s.curve, B[i])
		// sanity check
		if !s.curve.IsOnCurve(Bx, By) {
			return fmt.Errorf("Point A received from sender is not on curve: %s", s.curve.Params().Name)
		}

		// k0 = aB
		k0x, k0y := s.curve.ScalarMult(Bx, By, a)
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
		c <- m0

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
		c <- m1
	}

	// close channel
	close(c)

	return nil
}

func (s *simplest) Receive(choices []uint8, messages [][]byte, c chan []byte) error {
	if len(choices) != len(messages) || len(choices) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// Receive marshalled point A from sender
	A := <-c
	Ax, Ay := elliptic.Unmarshal(s.curve, A)
	// sanity check
	if !s.curve.IsOnCurve(Ax, Ay) {
		return fmt.Errorf("Point A received from sender is not on curve: %s", s.curve.Params().Name)
	}

	// Generate points B, 1 for each OT
	// TODO: should we store all B?
	B := make([][]byte, s.baseCount)
	bSecrets := make([][]byte, s.baseCount)
	for i := 0; i < s.baseCount; i++ {
		// generate receiver priv/pub key pairs
		b, Bx, By, err := elliptic.GenerateKey(s.curve, rand.Reader)
		bSecrets[i] = b
		if err != nil {
			return err
		}
		// for each choice bit, compute the resultant point B
		switch choices[i] {
		case 0:
			B[i] = elliptic.Marshal(s.curve, Bx, By)
		case 1:
			// B = A + bG
			Bx, By = s.curve.Add(Ax, Ay, Bx, By)
			B[i] = elliptic.Marshal(s.curve, Bx, By)
		default:
			return fmt.Errorf("Choice bits should be binary, got %v", choices[i])
		}

		// send marshalled point B to sender
		c <- B[i]
	}

	// receive encrypted messages
	var enc [][]byte
	for m := range c {
		enc = append(enc, m)
	}

	// close c
	close(c)

	//sanity check
	if len(enc) != s.baseCount {
		return fmt.Errorf("Miss match with # of elements in channel and baseCount, got %d elements", len(B))
	}

	// build keys for encypting messages
	for i := 0; i < s.baseCount; i++ {
		// right decryption key
		kx, ky := s.curve.ScalarMult(Ax, Ay, bSecrets[i])
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
		copy(messages[i], m)
	}

	return nil
}
