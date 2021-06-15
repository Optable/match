package ot

import (
	"crypt/rand"
	"fmt"
	"golang.org/x/crypto/curve25519"
)

type simplest struct {
	baseCount int
}

func NewSimplest(baseCount int) (*simplest, error) {
	return &simplest{baseCount: baseCount}, nil
}

func (s *simplest) Send(messages [][2]string, c chan []byte) error {
	if len(choices) != len(messages) || len(choices) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// generate sender secret public key pairs
	if a, err := genSecretKey(); err != nil {
		return err
	}
	if A, err := genPublicKey(a); err != nil {
		return err
	}

	// send point A to receiver
	c <- A

	// make a slice of point B, 1 for each OT
	return nil
}

func (s *simplest) Receive(choices []uint8, messages []string, c chan []byte) error {
	if len(choices) != len(messages) || len(choices) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// Receive point A from sender
	A := <-c

	// Generate points B, 1 for each OT
	B := make([]byte, s.baseCount)
	for i := 0; i < s.baseCount; i++ {
		if b, err := genSecretKey(); err != nil {
			return err
		}
		if B[i], err = genPublicKey(b); err != nil {
			return err
		}

		// for each choice bit, compute the resultant point B
	}

	// send points B to sender

	return nil
}
