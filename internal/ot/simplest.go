package ot

import (
	"crypto/elliptic"
	"fmt"
	"io"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

/*
1 out of 2 base OT
from the paper: The Simplest Protocol for Oblivious Transfer
by Tung Chou and Claudio Orlandi in 2015
Reference: https://eprint.iacr.org/2015/267.pdf
*/

type simplest struct {
	baseCount  int
	curve      elliptic.Curve
	encodeLen  int
	msgLen     []int
	cipherMode int
}

func newSimplest(baseCount int, curveName string, msgLen []int, cipherMode int) (simplest, error) {
	if len(msgLen) != baseCount {
		return simplest{}, ErrBaseCountMissMatch
	}
	curve, encodeLen := crypto.InitCurve(curveName)
	return simplest{baseCount: baseCount, curve: curve, encodeLen: encodeLen, msgLen: msgLen, cipherMode: cipherMode}, nil
}

func (s simplest) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	if len(messages) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	reader := crypto.NewECPointsReader(rw, s.encodeLen)
	writer := crypto.NewECPointsWriter(rw)

	// generate sender secret public key pairs
	secretA, pointA, err := crypto.GenerateKeyWithPoints(s.curve)
	if err != nil {
		return err
	}

	// send point A in marshaled []byte to receiver
	if err := writer.Write(pointA); err != nil {
		return err
	}

	// Precompute A = aA
	pointA = pointA.ScalarMult(secretA)

	// make a slice of point B, 1 for each OT, and receive them
	pointB := make([]crypto.Points, s.baseCount)
	for i := range pointB {
		pointB[i] = crypto.NewPoints(s.curve)
		if err := reader.Read(pointB[i]); err != nil {
			return err
		}
	}

	pointK := make([]crypto.Points, 2)
	var ciphertext []byte
	// encrypt plaintext messages and send it.
	for i := 0; i < s.baseCount; i++ {
		// k0 = aB
		pointK[0] = pointB[i].ScalarMult(secretA)
		//k1 = a(B - A) = aB - aA
		pointK[1] = pointK[0].Sub(pointA)

		// Encrypt plaintext message with key derived from received points B
		for choice, plaintext := range messages[i] {
			// encrypt plaintext using aes GCM mode
			ciphertext, err = crypto.Encrypt(s.cipherMode, pointK[choice].DeriveKeyFromECPoints(), uint8(choice), plaintext)
			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// send ciphertext
			if _, err = rw.Write(ciphertext); err != nil {
				return err
			}
		}
	}

	return
}

func (s simplest) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices)*8 != len(messages) || len(choices)*8 != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	reader := crypto.NewECPointsReader(rw, s.encodeLen)
	writer := crypto.NewECPointsWriter(rw)

	// Receive marshalled point A from sender
	pointA := crypto.NewPoints(s.curve)
	if err := reader.Read(pointA); err != nil {
		return err
	}

	// Generate points B, 1 for each OT
	bSecrets := make([][]byte, s.baseCount)
	var pointB crypto.Points
	for i := 0; i < s.baseCount; i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		bSecrets[i], pointB, err = crypto.GenerateKeyWithPoints(s.curve)
		if err != nil {
			return err
		}

		// for each choice bit, compute the resultant point B and send it
		if !util.BitSetInByte(choices, i) {
			// B
			if err := writer.Write(pointB); err != nil {
				return err
			}
		} else {
			// B = A + B
			if err := writer.Write(pointA.Add(pointB)); err != nil {
				return err
			}
		}
	}

	e := make([][]byte, 2)
	var pointK crypto.Points
	// receive encrypted messages, and decrypt it.
	for i := 0; i < s.baseCount; i++ {
		// compute # of bytes to be read.
		l := crypto.EncryptLen(s.cipherMode, s.msgLen[i])

		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// build keys for decrypting choice messages
		pointK = pointA.ScalarMult(bSecrets[i])

		// decrypt the message indexed by choice bit
		var bit byte
		if util.BitSetInByte(choices, i) {
			bit = 1
		}
		messages[i], err = crypto.Decrypt(s.cipherMode, pointK.DeriveKeyFromECPoints(), bit, e[bit])
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
	}

	return
}
