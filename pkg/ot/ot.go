package ot

import (
	"crypto/aes"
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

	nonceSize = 12 //aesgcm NonceSize
)

// OT implements different BaseOT
type Ot interface {
	Send(messages [][2][]byte, rw io.ReadWriter) error
	Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error
}

type Writer struct {
	w     io.Writer
	curve elliptic.Curve
}

type Reader struct {
	r         io.Reader
	curve     elliptic.Curve
	encodeLen int
}

type points struct {
	x *big.Int
	y *big.Int
}

func newWriter(w io.Writer, c elliptic.Curve) *Writer {
	return &Writer{w: w, curve: c}
}

func newReader(r io.Reader, c elliptic.Curve, l int) *Reader {
	return &Reader{r: r, curve: c, encodeLen: l}
}

func newPoints(x, y *big.Int) points {
	return points{x: x, y: y}
}

// Write writes the marshalled elliptic curve point to writer
func (w *Writer) write(p points) (err error) {
	if _, err = w.w.Write(elliptic.Marshal(w.curve, p.x, p.y)); err != nil {
		return err
	}
	return
}

// Read reads a marshalled elliptic curve point from reader and stores it in point
func (r *Reader) read(p points) (err error) {
	pt := make([]byte, r.encodeLen)
	if _, err = io.ReadFull(r.r, pt); err != nil {
		return err
	}

	px, py := elliptic.Unmarshal(r.curve, pt)

	p.x.Set(px)
	p.y.Set(py)
	return
}

// NewBaseOt returns an Ot of type t
func NewBaseOt(t int, baseCount int, curveName string, msgLen []int) (Ot, error) {
	switch t {
	case NaorPinkas:
		return newNaorPinkas(baseCount, curveName, msgLen)
	case Simplest:
		return newSimplest(baseCount, curveName, msgLen)
	default:
		return nil, ErrUnknownOt
	}
}

func initCurve(curveName string) (curve elliptic.Curve, encodeLen int) {
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
	encodeLen = len(elliptic.Marshal(curve, curve.Params().Gx, curve.Params().Gy))
	return
}

// deriveKey returns a key of 32 byte from an elliptic curve point
func deriveKey(point []byte) []byte {
	key := sha256.Sum256(point)
	return key[:]
}

// compute ciphertext length in bytes
func encryptLen(msgLen int) int {
	return nonceSize + aes.BlockSize + msgLen
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
