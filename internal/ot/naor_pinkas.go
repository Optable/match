package ot

import (
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
	// msgLen holds the length of each pairs of OT message
	// it serves to inform the receiver, how many bytes it is
	// expected to read
	msgLens []int
}

func NewNaorPinkas(msgLens []int) OT {
	return naorPinkas{msgLens: msgLens}
}

func (n naorPinkas) Send(otMessages []OTMessage, rw io.ReadWriter) (err error) {
	if len(n.msgLens) != len(otMessages) {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	reader := crypto.NewECPointReader(rw)
	writer := crypto.NewECPointWriter(rw)

	// generate sender point A w/o secret, since a is never used.
	_, pointA, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("error generating keys: %w", err)
	}

	// generate sender secret public key pairs used for encryption.
	secretR, pointR, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("error generating keys: %w", err)
	}

	// send point A to receiver
	if err := writer.Write(pointA); err != nil {
		return fmt.Errorf("error writing point: %w", err)
	}

	// send point R to receiver
	if err := writer.Write(pointR); err != nil {
		return fmt.Errorf("error writing point: %w", err)
	}

	// precompute A = rA
	pointA = pointA.ScalarMult(secretR)

	// encrypt plaintext messages and send them.
	for i := range otMessages {
		keyMaterial := crypto.NewPoint()
		// read keyMaterials to derive keys
		if err := reader.Read(keyMaterial); err != nil {
			return fmt.Errorf("error reading point: %w", err)
		}

		// compute and derive key for first OT message
		var keys [2]*crypto.Point
		// K0 = rK0
		keys[0] = keyMaterial.ScalarMult(secretR)
		// compute and derive key for second OT message
		// K1 = rA - rK0
		keys[1] = pointA.Sub(keys[0])

		// encrypt plaintext message with keys
		for choice, plaintext := range otMessages[i] {
			// encryption
			ciphertext := crypto.XorCipherWithBlake3(keys[choice].DeriveKeyFromECPoint(), uint8(choice), plaintext)

			// send ciphertext
			if _, err = rw.Write(ciphertext); err != nil {
				return fmt.Errorf("error writing bytes: %w", err)
			}
		}
	}

	return
}

func (n naorPinkas) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices)*8 != len(messages) || len(choices)*8 != len(n.msgLens) {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	reader := crypto.NewECPointReader(rw)
	writer := crypto.NewECPointWriter(rw)

	// receive point A from sender
	pointA := crypto.NewPoint()
	if err := reader.Read(pointA); err != nil {
		return fmt.Errorf("error reading point: %w", err)
	}
	// recieve point R from sender
	pointR := crypto.NewPoint()
	if err := reader.Read(pointR); err != nil {
		return fmt.Errorf("error reading point: %w", err)
	}

	for i := range messages {
		// generate receiver priv/pub key pairs going to take a long time.
		secretB, pointB, err := crypto.GenerateKey()
		if err != nil {
			return fmt.Errorf("error generating keys: %w", err)
		}

		// for each choice bit, compute the key material corresponding to
		// the choice bit and sent it.
		if !util.IsBitSet(choices, i) {
			// K0 = Kc = B
			if err := writer.Write(pointB); err != nil {
				return fmt.Errorf("error writing point: %w", err)
			}
		} else {
			// K1 = Kc = B
			// K0 = K1-c = A - B
			if err := writer.Write(pointA.Sub(pointB)); err != nil {
				return fmt.Errorf("error writing point: %w", err)
			}
		}

		// receive encrypted messages, and decrypt it.
		var encryptedOTMessages OTMessage
		// read both msg
		encryptedOTMessages[0] = make([]byte, n.msgLens[i])
		if _, err := io.ReadFull(rw, encryptedOTMessages[0]); err != nil {
			return fmt.Errorf("error reading bytes: %w", err)
		}

		encryptedOTMessages[1] = make([]byte, n.msgLens[i])
		if _, err := io.ReadFull(rw, encryptedOTMessages[1]); err != nil {
			return fmt.Errorf("error writing point: %w", err)
		}

		// build keys for decryption
		// K = bR
		pointK := pointR.ScalarMult(secretB)

		// decrypt the message indexed by choice bit
		choiceBit := util.BitExtract(choices, i)
		messages[i] = crypto.XorCipherWithBlake3(pointK.DeriveKeyFromECPoint(), choiceBit, encryptedOTMessages[choiceBit])
	}

	return
}
