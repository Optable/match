package ot

import (
	"crypto/elliptic"
	"fmt"
	"io"
	"math/big"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/cipher"
	"github.com/optable/match/internal/util"
)

/*
1 out of 2 base OT
from the paper: The Simplest Protocol for Oblivious Transfer
by Tung Chou and Claudio Orlandi in 2015
Reference: https://eprint.iacr.org/2015/267.pdf

Tested to be slightly faster than Naor-Pinkas
but has the same computation costs.
*/

type simplestBitSet struct {
	baseCount  int
	curve      elliptic.Curve
	encodeLen  int
	msgLen     []int
	cipherMode int
}

func newSimplestBitSet(baseCount int, curveName string, msgLen []int, cipherMode int) (simplestBitSet, error) {
	if len(msgLen) != baseCount {
		return simplestBitSet{}, ErrBaseCountMissMatch
	}
	curve, encodeLen := initCurve(curveName)
	return simplestBitSet{baseCount: baseCount, curve: curve, encodeLen: encodeLen, msgLen: msgLen, cipherMode: cipherMode}, nil
}

func (s simplestBitSet) Send(messages [][]*bitset.BitSet, rw io.ReadWriter) (err error) {
	if len(messages) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// Instantiate Reader, Writer
	reader := newReader(rw, s.curve, s.encodeLen)
	writer := newWriter(rw, s.curve)

	// generate sender secret public key pairs
	a, A, err := generateKeyWithPoints(s.curve)
	if err != nil {
		return err
	}

	// send point A in marshaled []byte to receiver
	if err := writer.write(A); err != nil {
		return err
	}

	// Precompute A = aA
	A = A.scalarMult(a)

	// make a slice of point B, 1 for each OT, and receive them
	B := make([]points, s.baseCount)
	for i := range B {
		B[i] = newPoints(s.curve, new(big.Int), new(big.Int))
		if err := reader.read(B[i]); err != nil {
			return err
		}
	}

	K := make([]points, 2)
	// encrypt plaintext messages and send it.
	for i := 0; i < s.baseCount; i++ {
		// sanity check
		if !B[i].isOnCurve() {
			return fmt.Errorf("point A received from sender is not on curve: %s", s.curve.Params().Name)
		}

		// k0 = aB
		K[0] = B[i].scalarMult(a)
		//k1 = a(B - A) = aB - aA
		K[1] = K[0].sub(A)
		// Encrypt plaintext message with key derived from received points B
		for choice, plaintext := range messages[i] {
			// encrypt plaintext using aes GCM mode
			// ciphertext, err := cipher.Encrypt(s.cipherMode, K[choice].deriveKey(), uint8(choice), util.BitSetToBits(plaintext))
			ciphertext, err := cipher.Encrypt(s.cipherMode, K[choice].deriveKey(), uint8(choice), util.BitSetToBytes(plaintext))
			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// send ciphertext
			if _, err = util.BytesToBitSet(ciphertext).WriteTo(rw); err != nil {
				return err
			}
		}
		/*
			// Encrypt plaintext message with key derived from received points B
			for choice, plaintext := range messages[i] {
				// encrypt plaintext using aes GCM mode
				// ciphertext, err := cipher.Encrypt(s.cipherMode, K[choice].deriveKey(), uint8(choice), util.BitSetToBits(plaintext))
				ciphertext, err := cipher.XorCipherWithBlake3BitSet(K[choice].deriveKey(), uint8(choice), plaintext)
				if err != nil {
					return fmt.Errorf("error encrypting sender message: %s", err)
				}

				// send ciphertext
				if _, err = ciphertext.WriteTo(rw); err != nil {
					return err
				}
			}
		*/
	}

	return
}

func (s simplestBitSet) Receive(choices *bitset.BitSet, messages []*bitset.BitSet, rw io.ReadWriter) (err error) {
	if int(choices.Len()) < len(messages) || int(choices.Len()) > len(messages)+63 || len(messages) != s.baseCount {
		return ErrBaseCountMissMatch
	}

	// instantiate Reader, Writer
	reader := newReader(rw, s.curve, s.encodeLen)
	writer := newWriter(rw, s.curve)

	// Receive marshalled point A from sender
	A := newPoints(s.curve, new(big.Int), new(big.Int))
	if err := reader.read(A); err != nil {
		return err
	}

	// sanity check
	if !A.isOnCurve() {
		return fmt.Errorf("point A received from sender is not on curve: %s", s.curve.Params().Name)
	}

	// Generate points B, 1 for each OT
	bSecrets := make([][]byte, s.baseCount)
	var B points
	var b []byte
	for i := uint(0); i < uint(s.baseCount); i++ {
		// generate receiver priv/pub key pairs going to take a long time.
		b, B, err = generateKeyWithPoints(s.curve)
		if err != nil {
			return err
		}
		bSecrets[i] = b

		// for each choice bit, compute the resultant point B and send it
		if choices.Test(i) {
			// B = A + B
			if err := writer.write(A.add(B)); err != nil {
				return err
			}
		} else {
			// B
			if err := writer.write(B); err != nil {
				return err
			}
		}
	}

	e := make([]*bitset.BitSet, 2)
	var K points
	// receive encrypted messages, and decrypt it.
	for i := 0; i < s.baseCount; i++ {
		// compute # of bytes to be read.
		l := uint(cipher.EncryptLen(s.cipherMode, s.msgLen[i]))

		// read both msg
		for j := range e {
			e[j] = bitset.New(l)
			if _, err = e[j].ReadFrom(rw); err != nil {
				return err
			}
		}

		// build keys for decrypting choice messages
		K = A.scalarMult(bSecrets[i])
		// decrypt the message indexed by choice bit
		var choice uint8
		if choices.Test(uint(i)) {
			choice = 1
		}

		// message, err := cipher.Decrypt(s.cipherMode, K.deriveKey(), choice, util.BitSetToBits(e[choice]))
		message, err := cipher.Decrypt(s.cipherMode, K.deriveKey(), choice, util.BitSetToBytes(e[choice]))
		if err != nil {
			return fmt.Errorf("error decrypting sender message: %s", err)
		}
		messages[i] = util.BytesToBitSet(message)
		// message, err := cipher.Decrypt(s.cipherMode, K.deriveKey(), choice, util.BitSetToBits(e[choice]))
		/*
			if choices.Test(uint(i)) {
				messages[i], err = cipher.XorCipherWithBlake3BitSet(K.deriveKey(), 1, e[1])
			} else {
				messages[i], err = cipher.XorCipherWithBlake3BitSet(K.deriveKey(), 0, e[0])
			}
			if err != nil {
				return fmt.Errorf("error decrypting sender message: %s", err)
			}
		*/
	}

	return
}
