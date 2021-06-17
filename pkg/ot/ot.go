package ot

import (
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
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
	Send(messages [][2][]byte, rw io.ReadWriter) error
	Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error
}

type Writer struct {
	w io.Writer
}

type Reader struct {
	r io.Reader
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Write writes the marshalled elliptic curve point to writer
func (w *Writer) Write(point []byte) (err error) {
	if _, err = w.w.Write(point); err != nil {
		return err
	}
	return
}

// Read reads a marshalled elliptic curve point from reader and stores it in point
func (r *Reader) Read(point []byte) (err error) {
	if _, err = r.r.Read(point); err != nil {
		return err
	}
	return
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
	case "P224":
		curve = elliptic.P224()
	case "P256":
		curve = elliptic.P256()
	case "P384":
		curve = elliptic.P384()
	case "P521":
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
