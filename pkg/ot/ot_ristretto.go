package ot

import (
	"crypto/sha256"
	"io"

	gr "github.com/bwesterb/go-ristretto"
)

var encodeLen = 32 //ristretto point encoded length, as well as aes key

type writerRistretto struct {
	w io.Writer
}

type readerRistretto struct {
	r io.Reader
}

func newWriterRistretto(w io.Writer) *writerRistretto {
	return &writerRistretto{w: w}
}

func newReaderRistretto(r io.Reader) *readerRistretto {
	return &readerRistretto{r: r}
}

// Write writes the marshalled elliptic curve point to writer
func (w *writerRistretto) write(p *gr.Point) (err error) {
	pByte, err := p.MarshalBinary()
	if err != nil {
		return err
	}

	if _, err = w.w.Write(pByte); err != nil {
		return err
	}
	return
}

// Read reads a marshalled elliptic curve point from reader and stores it in point
func (r *readerRistretto) read(p *gr.Point) (err error) {
	pt := make([]byte, encodeLen)
	if _, err = io.ReadFull(r.r, pt); err != nil {
		return err
	}

	if err := p.UnmarshalBinary(pt); err != nil {
		return err
	}
	return
}

// NewBaseOt returns an Ot of type t
func NewBaseOtRistretto(t int, baseCount int, msgLen []int) (Ot, error) {
	switch t {
	case NaorPinkas:
		return newNaorPinkasRistretto(baseCount, msgLen)
	case Simplest:
		return newSimplestRistretto(baseCount, msgLen)
	default:
		return nil, ErrUnknownOt
	}
}

// generatekeys returns a secret key scalar
// and a public key ristretto point
func generateKeys() (secretKey gr.Scalar, publicKey gr.Point) {
	secretKey.Rand()
	publicKey.ScalarMultBase(&secretKey)

	return
}

// deriveKey returns a key of 32 byte from an elliptic curve point
func deriveKeyRistretto(point *gr.Point) ([]byte, error) {
	buf, err := point.MarshalBinary()
	if err != nil {
		return nil, err
	}

	key := sha256.Sum256(buf)
	return key[:], nil
}
