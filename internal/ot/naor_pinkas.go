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
from the paper: Efficient Oblivious Transfer Protocol
by Moni Naor and Benny Pinkas in 2001.
reference: https://dl.acm.org/doi/abs/10.5555/365411.365502
*/

type naorPinkas struct {
	baseCount  int
	curve      elliptic.Curve
	encodeLen  int
	msgLen     []int
	cipherMode int
}

func newNaorPinkas(baseCount int, curveName string, msgLen []int, cipherMode int) (naorPinkas, error) {
	if len(msgLen) != baseCount {
		return naorPinkas{}, ErrBaseCountMissMatch
	}
	curve, encodeLen := crypto.InitCurve(curveName)
	return naorPinkas{baseCount: baseCount, curve: curve, encodeLen: encodeLen, msgLen: msgLen, cipherMode: cipherMode}, nil
}

func (n naorPinkas) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	if len(messages) != n.baseCount {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	reader := newReader(rw, n.encodeLen)
	writer := newWriter(rw)

	// generate sender point A w/o secret, since a is never used.
	_, pointA, err := crypto.GenerateKeyWithPoints(n.curve)
	if err != nil {
		return err
	}

	// generate sender secret public key pairs used for encryption.
	secretR, pointR, err := crypto.GenerateKeyWithPoints(n.curve)
	if err != nil {
		return err
	}

	// send point A to receiver
	if err := writer.write(pointA); err != nil {
		return err
	}

	// send point R to receiver
	if err := writer.write(pointR); err != nil {
		return err
	}

	// precompute A = rA
	pointA = pointA.ScalarMult(secretR)

	// make a slice of points to receive K0.
	pointK0 := make([]crypto.Points, n.baseCount)
	for i := range pointK0 {
		pointK0[i] = crypto.NewPoints(n.curve)
		if err := reader.read(pointK0[i]); err != nil {
			return err
		}
	}

	pointK := make([]crypto.Points, 2)
	var ciphertext []byte
	// encrypt plaintext messages and send them.
	for i := 0; i < n.baseCount; i++ {
		// compute K0 = rK0
		pointK[0] = pointK0[i].ScalarMult(secretR)
		// compute K1 = rA - rK0
		pointK[1] = pointA.Sub(pointK[0])

		// encrypt plaintext message with key derived from K0, K1
		for choice, plaintext := range messages[i] {
			// encryption
			ciphertext, err = crypto.Encrypt(n.cipherMode, pointK[choice].DeriveKey(), uint8(choice), plaintext)
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

func (n naorPinkas) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices)*8 != len(messages) || len(choices)*8 != n.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	reader := newReader(rw, n.encodeLen)
	writer := newWriter(rw)

	// receive point A from sender
	pointA := crypto.NewPoints(n.curve)
	if err := reader.read(pointA); err != nil {
		return err
	}

	// recieve point R from sender
	pointR := crypto.NewPoints(n.curve)
	if err := reader.read(pointR); err != nil {
		return err
	}

	// Generate points B, 1 for each OT
	bSecrets := make([][]byte, n.baseCount)
	var pointB crypto.Points
	for i := 0; i < n.baseCount; i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		bSecrets[i], pointB, err = crypto.GenerateKeyWithPoints(n.curve)
		if err != nil {
			return err
		}

		// for each choice bit, compute the resultant point Kc, K1-c and send K0
		if util.TestBitSetInByte(choices, i) == 0 {
			// K0 = Kc = B
			if err := writer.write(pointB); err != nil {
				return err
			}
		} else {
			// K1 = Kc = B
			// K0 = K1-c = A - B
			if err := writer.write(pointA.Sub(pointB)); err != nil {
				return err
			}
		}
	}

	e := make([][]byte, 2)
	var pointK crypto.Points
	// receive encrypted messages, and decrypt it.
	for i := 0; i < n.baseCount; i++ {
		// compute # of bytes to be read.
		l := crypto.EncryptLen(n.cipherMode, n.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err := io.ReadFull(reader.r, e[j]); err != nil {
				return err
			}
		}

		// build keys for decryption
		// K = bR
		pointK = pointR.ScalarMult(bSecrets[i])

		// decrypt the message indexed by choice bit
		bit := util.TestBitSetInByte(choices, i)
		messages[i], err = crypto.Decrypt(n.cipherMode, pointK.DeriveKey(), bit, e[bit])
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
	}

	return
}
