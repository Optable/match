package ot

import (
	"crypto/elliptic"
	"fmt"
	"io"
	"math/big"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/util"
)

/*
1 out of 2 base OT
from the paper: Efficient Oblivious Transfer Protocol
by Moni Naor and Benny Pinkas in 2001.
reference: https://dl.acm.org/doi/abs/10.5555/365411.365502

Naor-Pinkas OT is used in most papers, but it is slightly slower than Simplest OT.
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
	curve, encodeLen := initCurve(curveName)
	return naorPinkas{baseCount: baseCount, curve: curve, encodeLen: encodeLen, msgLen: msgLen, cipherMode: cipherMode}, nil
}

func (n naorPinkas) Send(messages [][]*bitset.BitSet, rw io.ReadWriter) (err error) {
	//func (n naorPinkas) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	if len(messages) != n.baseCount {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	reader := newReader(rw, n.curve, n.encodeLen)
	writer := newWriter(rw, n.curve)

	// generate sender point A w/o secret, since a is never used.
	_, A, err := generateKeyWithPoints(n.curve)
	if err != nil {
		return err
	}

	// generate sender secret public key pairs  used for encryption.
	r, R, err := generateKeyWithPoints(n.curve)
	if err != nil {
		return err
	}

	// send point A to receiver
	if err := writer.write(A); err != nil {
		return err
	}
	// send point R to receiver
	if err := writer.write(R); err != nil {
		return err
	}

	// precompute A = rA
	A = A.scalarMult(r)

	// make a slice of points to receive K0.
	pointK0 := make([]points, n.baseCount)
	for i := range pointK0 {
		pointK0[i] = newPoints(n.curve, new(big.Int), new(big.Int))
		if err := reader.read(pointK0[i]); err != nil {
			return err
		}
	}

	K := make([]points, 2)
	var ciphertext []byte
	// encrypt plaintext messages and send them.
	for i := 0; i < n.baseCount; i++ {
		// sanity check
		if !pointK0[i].isOnCurve() {
			return fmt.Errorf("point A received from sender is not on curve: %s", n.curve.Params().Name)
		}

		// compute K0 = rK0
		K[0] = pointK0[i].scalarMult(r)
		// compute K1 = rA - rK0
		K[1] = A.sub(K[0])

		// encrypt plaintext message with key derived from K0, K1
		for choice, plaintext := range messages[i] {
			// encryption
			ciphertext, err = encrypt(n.cipherMode, K[choice].deriveKey(), uint8(choice), util.BitSetToBytes(plaintext))
			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// convert ciphertext into BitSet
			cipherBitSet := util.BytesToBitSet(ciphertext)

			// send ciphertext
			// TODO check that rw is the proper stream to write to
			if _, err = cipherBitSet.WriteTo(rw); err != nil {
				return err
			}
		}
	}

	return
}

//func (n naorPinkas) Receive(choices *bitset.BitSet, messages [][]byte, rw io.ReadWriter) (err error) {
func (n naorPinkas) Receive(choices *bitset.BitSet, messages []*bitset.BitSet, rw io.ReadWriter) (err error) {
	if int(choices.Len()) < len(messages) || int(choices.Len()) > len(messages)+63 || len(messages) != n.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	reader := newReader(rw, n.curve, n.encodeLen)
	writer := newWriter(rw, n.curve)

	// receive point A from sender
	A := newPoints(n.curve, new(big.Int), new(big.Int))
	if err := reader.read(A); err != nil {
		return err
	}

	// receive point R from sender
	R := newPoints(n.curve, new(big.Int), new(big.Int))
	if err := reader.read(R); err != nil {
		return err
	}

	// sanity check
	if !A.isOnCurve() || !R.isOnCurve() {
		return fmt.Errorf("points received from sender is not on curve: %s", n.curve.Params().Name)
	}

	// Generate points B, 1 for each OT
	bSecrets := make([][]byte, n.baseCount)
	var B points
	var b []byte
	for i := uint(0); i < uint(n.baseCount); i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		b, B, err = generateKeyWithPoints(n.curve)
		if err != nil {
			return err
		}
		bSecrets[i] = b

		// for each choice bit, compute the resultant point Kc, K1-c and send K0
		if choices.Test(i) {
			// K1 = Kc = B
			// K0 = K1-c = A - B
			if err := writer.write(A.sub(B)); err != nil {
				return err
			}
		} else {
			// K0 = Kc = B
			if err := writer.write(B); err != nil {
				return err
			}
		}
	}

	e := make([]*bitset.BitSet, 2)
	var K points
	// receive encrypted messages (as BitSets) and decrypt them
	for i := uint(0); i < uint(n.baseCount); i++ {
		// read both messages
		for j := range e {
			e[j] = bitset.New(8)
			if _, err := e[j].ReadFrom(rw); err != nil {
				return err
			}
		}

		// build keys for decryption
		// K = bR
		K = R.scalarMult(bSecrets[i])

		// decrypt the message indexed by choice bit
		var choice uint8
		if choices.Test(i) {
			choice = 1
		}
		message, err := decrypt(n.cipherMode, K.deriveKey(), choice, util.BitSetToBytes(e[choice]))
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
		messages[i] = util.BytesToBitSet(message)
	}

	return
}
