package crypto

import (
	"io"

	gr "github.com/bwesterb/go-ristretto"
	"github.com/zeebo/blake3"
)

const encodeLen = 32 //ristretto point encoded length, as well as aes key

type ristrettoWriter struct {
	w io.Writer
}

type ristrettoReader struct {
	r io.Reader
}

func NewRistrettoWriter(w io.Writer) *ristrettoWriter {
	return &ristrettoWriter{w: w}
}

func NewRistrettoReader(r io.Reader) *ristrettoReader {
	return &ristrettoReader{r: r}
}

// Write writes the marshalled elliptic curve point to writer
func (w *ristrettoWriter) Write(p *gr.Point) (err error) {
	pByte, err := p.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = w.w.Write(pByte)
	return err
}

// Read reads a marshalled elliptic curve point from reader and stores it in point
func (r *ristrettoReader) Read(p *gr.Point) (err error) {
	pt := make([]byte, encodeLen)
	if _, err = io.ReadFull(r.r, pt); err != nil {
		return err
	}

	return p.UnmarshalBinary(pt)
}

// GenerateRistrettoKeys returns a secret key scalar
// and a public key ristretto point
func GenerateRistrettoKeys() (secretKey gr.Scalar, publicKey gr.Point) {
	secretKey.Rand()
	publicKey.ScalarMultBase(&secretKey)

	return
}

func GeneratePublicRistrettoKey() (publicKey gr.Point) {
	var p gr.Point
	p.Rand()
	return p
}

// hashToKey returns a key of 32 byte from an elliptic curve point
func hashToKey(point []byte) []byte {
	key := blake3.Sum256(point)
	return key[:]
}

// DeriveRistrettoKeys returns a key of 32 byte from an elliptic curve point
func DeriveRistrettoKey(point *gr.Point) ([]byte, error) {
	buf, err := point.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return hashToKey(buf), nil
}
