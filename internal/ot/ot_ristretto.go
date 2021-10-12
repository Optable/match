package ot

import (
	"io"

	gr "github.com/bwesterb/go-ristretto"
	"github.com/optable/match/internal/crypto"
)

/*
OT interface
*/

const encodeLen = 32 //ristretto point encoded length, as well as aes key

type ristrettoWriter struct {
	w io.Writer
}

type ristrettoReader struct {
	r io.Reader
}

func newRistrettoWriter(w io.Writer) *ristrettoWriter {
	return &ristrettoWriter{w: w}
}

func newRistrettoReader(r io.Reader) *ristrettoReader {
	return &ristrettoReader{r: r}
}

// Write writes the marshalled elliptic curve point to writer
func (w *ristrettoWriter) write(p *gr.Point) (err error) {
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
func (r *ristrettoReader) read(p *gr.Point) (err error) {
	pt := make([]byte, encodeLen)
	if _, err = io.ReadFull(r.r, pt); err != nil {
		return err
	}

	if err := p.UnmarshalBinary(pt); err != nil {
		return err
	}
	return
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

	return crypto.DeriveKey(buf), nil
}
