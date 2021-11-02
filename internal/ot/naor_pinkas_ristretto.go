package ot

import (
	"fmt"
	"io"

	gr "github.com/bwesterb/go-ristretto"
	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

/*
1 out of 2 base OT
from the paper: Efficient Oblivious Transfer Protocol
by Moni Naor and Benny Pinkas in 2001.
reference: https://dl.acm.org/doi/abs/10.5555/365411.365502

Naor-Pinkas OT implemented using Ristretto points for the elliptic curve operations.
*/

type naorPinkasRistretto struct {
	baseCount  int
	msgLen     []int
	cipherMode crypto.CipherMode
}

func newNaorPinkasRistretto(baseCount int, msgLen []int, cipherMode crypto.CipherMode) (naorPinkasRistretto, error) {
	if len(msgLen) != baseCount {
		return naorPinkasRistretto{}, ErrBaseCountMissMatch
	}
	return naorPinkasRistretto{baseCount: baseCount, msgLen: msgLen, cipherMode: cipherMode}, nil
}

func (n naorPinkasRistretto) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	if len(messages) != n.baseCount {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	reader := crypto.NewRistrettoReader(rw)
	writer := crypto.NewRistrettoWriter(rw)

	// generate sender A point w/o secret, since a is never used.
	var pointA = crypto.GeneratePublicRistrettoKey()

	// generate sender secret public key pairs used for encryption
	secretR, pointR := crypto.GenerateRistrettoKeys()

	// send both public keys to receiver
	if err := writer.Write(&pointA); err != nil {
		return err
	}
	if err := writer.Write(&pointR); err != nil {
		return err
	}

	// precompute A = rA
	pointA.ScalarMult(&pointA, &secretR)

	// make a slice of ristretto points to receive K0.
	pointK0 := make([]gr.Point, n.baseCount)
	for i := range pointK0 {
		if err := reader.Read(&pointK0[i]); err != nil {
			return err
		}
	}

	pointK := make([]gr.Point, 2)
	// encrypt plaintext message and send them.
	for i := 0; i < n.baseCount; i++ {
		// compute K0 = rK0
		pointK[0].ScalarMult(&pointK0[i], &secretR)
		// compute K1 = rA - rK0
		pointK[1].Sub(&pointA, &pointK[0])

		// encrypt plaintext message with key derived from K0, K1
		for choice, plaintext := range messages[i] {
			// derive key for encryption
			key, err := crypto.DeriveRistrettoKey(&pointK[choice])
			if err != nil {
				return err
			}

			// encrypt
			ciphertext, err := crypto.Encrypt(n.cipherMode, key, uint8(choice), plaintext)
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

func (n naorPinkasRistretto) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices)*8 != len(messages) || len(choices)*8 != n.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	reader := crypto.NewRistrettoReader(rw)
	writer := crypto.NewRistrettoWriter(rw)

	// Receive point A from sender
	var pointA gr.Point
	if err := reader.Read(&pointA); err != nil {
		return err
	}

	// Receive point R from sender
	var pointR gr.Point
	if err := reader.Read(&pointR); err != nil {
		return err
	}

	// Generate points B, 1 for each OT,
	bSecrets := make([]gr.Scalar, n.baseCount)
	var pointB gr.Point
	for i := 0; i < n.baseCount; i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		bSecrets[i], pointB = crypto.GenerateRistrettoKeys()

		// for each choice bit, compute the resultant point Kc, K1-c and send K0
		if !util.BitSetInByte(choices, i) {
			// K0 = Kc = B
			// K1 = K1-c = A - B
			if err := writer.Write(&pointB); err != nil {
				return err
			}
		} else {
			// K1 = Kc = B
			// K0 = K1-c = A - B
			pointB.Sub(&pointA, &pointB)
			if err := writer.Write(&pointB); err != nil {
				return err
			}
		}
	}

	e := make([][]byte, 2)
	var pointK gr.Point
	// receive encrypted messages, and decrypt it.
	for i := 0; i < n.baseCount; i++ {
		// compute # of bytes to be read.
		l := crypto.EncryptLen(n.cipherMode, n.msgLen[i])
		// read both msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err := io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// build keys for decrypting choice messages
		// K = bR
		pointK.ScalarMult(&pointR, &bSecrets[i])
		key, err := crypto.DeriveRistrettoKey(&pointK)
		if err != nil {
			return err
		}

		// decrypt the message indexed by choice bit
		var bit byte
		if util.BitSetInByte(choices, i) {
			bit = 1
		}
		messages[i], err = crypto.Decrypt(n.cipherMode, key, bit, e[bit])
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
	}

	return
}
