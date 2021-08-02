package ot

import (
	"crypto/elliptic"
	"crypto/sha256"
	"fmt"
	"io"
)

const (
	NaorPinkas = iota
	Simplest
)

var (
	ErrUnknownOt           = fmt.Errorf("cannot create an Ot that follows an unknown protocol")
	ErrBaseCountMissMatch  = fmt.Errorf("provided slices is not the same length as the number of base OT")
	ErrByteLengthMissMatch = fmt.Errorf("provided bytes do not have the same length for XOR operations")
	ErrEmptyMessage        = fmt.Errorf("attempt to perform OT on empty messages")

	nonceSize = 12 //aesgcm NonceSize
)

// OT implements different BaseOT
type Ot interface {
	Send(messages [][2][]byte, rw io.ReadWriter) error
	Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error
}

type writer struct {
	w     io.Writer
	curve elliptic.Curve
}

type reader struct {
	r         io.Reader
	curve     elliptic.Curve
	encodeLen int
}

func newWriter(w io.Writer, c elliptic.Curve) *writer {
	return &writer{w: w, curve: c}
}

func newReader(r io.Reader, c elliptic.Curve, l int) *reader {
	return &reader{r: r, curve: c, encodeLen: l}
}

// Write writes the marshalled elliptic curve point to writer
func (w *writer) write(p points) (err error) {
	if _, err = w.w.Write(elliptic.Marshal(w.curve, p.x, p.y)); err != nil {
		return err
	}
	return
}

// Read reads a marshalled elliptic curve point from reader and stores it in point
func (r *reader) read(p points) (err error) {
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
func NewBaseOt(t int, ristretto bool, baseCount int, curveName string, msgLen []int, cipherMode int) (Ot, error) {
	switch t {
	case NaorPinkas:
		if ristretto {
			return newNaorPinkasRistretto(baseCount, msgLen, cipherMode)
		}
		return newNaorPinkas(baseCount, curveName, msgLen, cipherMode)
	case Simplest:
		if ristretto {
			return newSimplestRistretto(baseCount, msgLen, cipherMode)
		}
		return newSimplest(baseCount, curveName, msgLen, cipherMode)
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
