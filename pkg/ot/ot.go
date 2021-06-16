package ot

import (
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

const (
	NaorPinkas = iota
	Simplest
)

var (
	ErrUnknownOt          = fmt.Errorf("cannot create an Ot that follows an unknown protocol")
	ErrBaseCountMissMatch = fmt.Errorf("provided slices is not the same length as the number of base OT.")

	ZeroByte = big.NewInt(0).Bytes()
	OneByte  = big.NewInt(1).Bytes()
)

// OT implements different BaseOT
type Ot interface {
	Send(messages [][2][]byte, c chan []byte) error
	Receive(choices []uint8, messages [][]byte, c chan []byte) error
}

// NewBaseOt returns an Ot of type t
func NewBaseOt(t int, baseCount int, curveName string) (Ot, error) {
	switch t {
	case NaorPinkas:
		return NewNaorPinkas(baseCount, curveName)
	case Simplest:
		return NewSimplest(baseCount, curveName)
	default:
		return nil, ErrUnknownOt
	}
}

func InitCurve(curveName string) (curve elliptic.Curve) {
	switch curveName {
	case "p224":
		curve = elliptic.P224()
	case "P256":
		curve = elliptic.P256()
	case "P384":
		curve = elliptic.P384()
	case "p521":
		curve = elliptic.P521()
	default:
		curve = elliptic.P256()
	}
	return
}

// deriveKey returns a key of 32 byte from an elliptic curve point
func deriveKey(point []byte) []byte {
	key := sha256.Sum256(point)
	return key[:]
}

// aes GCM block cipher encryption
func encrypt(block cipher.Block, plainText []byte) (cipherText []byte, err error) {
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// encrypted cipher text is appended after nonce
	cipherText = aesgcm.Seal(nonce, nonce, plainText, nil)
	return
}

func decrypt(block cipher.Block, cipherText []byte) (plainText []byte, err error) {
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	nonce, enc := cipherText[:nonceSize], cipherText[nonceSize:]

	if plainText, err = aesgcm.Open(nil, nonce, enc, nil); err != nil {
		return nil, err
	}
	return
}
