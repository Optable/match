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
	msgLen []int
}

func NewNaorPinkas(msgLen []int) (OT, error) {
	return naorPinkas{msgLen: msgLen}, nil
}

func (n naorPinkas) Send(otMessages []OTMessage, rw io.ReadWriter) (err error) {
	if len(n.msgLen) != len(otMessages) {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	reader := crypto.NewECPointReader(rw)
	writer := crypto.NewECPointWriter(rw)

	// generate sender point A w/o secret, since a is never used.
	_, pointA, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	// generate sender secret public key pairs used for encryption.
	secretR, pointR, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	// send point A to receiver
	if err := writer.Write(pointA); err != nil {
		return err
	}

	// send point R to receiver
	if err := writer.Write(pointR); err != nil {
		return err
	}

	// precompute A = rA
	pointA = pointA.ScalarMult(secretR)

	var ciphertext []byte
	// encrypt plaintext messages and send them.
	for i := range otMessages {
		// receive key material
		keyMaterial := crypto.NewPoint()
		if err := reader.Read(keyMaterial); err != nil {
			return err
		}
		var keys [2]*crypto.Point
		// compute K0 = rK0
		keys[0] = keyMaterial.ScalarMult(secretR)
		// compute K1 = rA - rK0
		keys[1] = pointA.Sub(keys[0])

		// encrypt plaintext message with key derived from K0, K1
		for choice, plaintext := range otMessages[i] {
			// encryption
			ciphertext, err = crypto.XorCipherWithBlake3(keys[choice].DeriveKeyFromECPoint(), uint8(choice), plaintext)
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

func (n naorPinkas) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices)*8 != len(messages) || len(choices)*8 != len(n.msgLen) {
		return ErrBaseCountMissMatch
	}
	// instantiate Reader, Writer
	reader := crypto.NewECPointReader(rw)
	writer := crypto.NewECPointWriter(rw)
	// receive point A from sender
	pointA := crypto.NewPoint()
	if err := reader.Read(pointA); err != nil {
		return err
	}
	// recieve point R from sender
	pointR := crypto.NewPoint()
	if err := reader.Read(pointR); err != nil {
		return err
	}
	// Generate points B, 1 for each OT
	bSecrets := make([][]byte, len(messages))
	var pointB *crypto.Point
	for i := range messages {
		// generate receiver priv/pub key pairs going to take a long time.
		bSecrets[i], pointB, err = crypto.GenerateKey()
		if err != nil {
			return err
		}

		// for each choice bit, compute the resultant point Kc, K1-c and send K0
		if !util.BitSetInByte(choices, i) {
			// K0 = Kc = B
			if err := writer.Write(pointB); err != nil {
				return err
			}
		} else {
			// K1 = Kc = B
			// K0 = K1-c = A - B
			if err := writer.Write(pointA.Sub(pointB)); err != nil {
				return err
			}
		}
	}

	e := make([][]byte, 2)
	var pointK *crypto.Point
	// receive encrypted messages, and decrypt it.
	for i := range messages {
		// read both msg
		for j := range e {
			e[j] = make([]byte, n.msgLen[i])
			if _, err := io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// build keys for decryption
		// K = bR
		pointK = pointR.ScalarMult(bSecrets[i])

		// decrypt the message indexed by choice bit
		var bit byte
		if util.BitSetInByte(choices, i) {
			bit = 1
		}
		messages[i], err = crypto.XorCipherWithBlake3(pointK.DeriveKeyFromECPoint(), bit, e[bit])
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
	}

	return
}
