package ot

import (
	"fmt"
	"io"

	"github.com/optable/match/internal/crypto"
)

/*
OT interface
*/

const (
	// enumerated base OTs
	NaorPinkas = iota
	Simplest
)

var (
	ErrUnknownOT          = fmt.Errorf("cannot create an Ot that follows an unknown protocol")
	ErrBaseCountMissMatch = fmt.Errorf("provided slices is not the same length as the number of base OT")
	ErrEmptyMessage       = fmt.Errorf("attempt to perform OT on empty messages")
)

// OT implements different BaseOT
type OT interface {
	Send(messages [][][]byte, rw io.ReadWriter) error
	Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error
}

// NewBaseOT returns an OT of type t
func NewBaseOT(t int, ristretto bool, baseCount int, curveName string, msgLen []int, cipherMode int) (OT, error) {
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
		return nil, ErrUnknownOT
	}
}

// writer for elliptic curve points
type writer struct {
	w io.Writer
}

// reader for elliptic curve points
type reader struct {
	r         io.Reader
	encodeLen int
}

// newWriter returns an elliptic curve point writer
func newWriter(w io.Writer) *writer {
	return &writer{w: w}
}

// newReader returns an elliptic curve point reader
func newReader(r io.Reader, l int) *reader {
	return &reader{r: r, encodeLen: l}
}

// write writes the marshalled elliptic curve point to writer
func (w *writer) write(p crypto.Points) (err error) {
	_, err = w.w.Write(p.Marshal())
	return err
}

// read reads a marshalled elliptic curve point from reader and stores it in point
func (r *reader) read(p crypto.Points) (err error) {
	pt := make([]byte, r.encodeLen)
	if _, err = io.ReadFull(r.r, pt); err != nil {
		return err
	}

	return p.Unmarshal(pt)
}
