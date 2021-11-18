package crypto

import (
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"

	"github.com/zeebo/blake3"
)

/*
High level api for operating on P256 elliptic curve Points.
*/

var (
	curve     = elliptic.P256()
	encodeLen = encodeLenWithCurve(curve)
)

// LenEncodeOnCurve returns the number of bytes needed to encode a point
// on the curve
func encodeLenWithCurve(curve elliptic.Curve) int {
	return len(elliptic.MarshalCompressed(curve, curve.Params().Gx, curve.Params().Gy))
}

// Point represents a point on an elliptic curve
type Point struct {
	x *big.Int
	y *big.Int
}

// NewPoint returns a Point
func NewPoint() *Point {
	return &Point{x: new(big.Int), y: new(big.Int)}
}

// Marshal converts a Point to a byte slice representation
func (p *Point) Marshal() []byte {
	return elliptic.MarshalCompressed(curve, p.x, p.y)
}

// Unmarshal takes in a marshaledPoint byte slice and extracts the Point object
func (p *Point) Unmarshal(marshaledPoint []byte) error {
	x, y := elliptic.UnmarshalCompressed(curve, marshaledPoint)

	// on error of Unmarshal, x is nil
	if x == nil {
		return fmt.Errorf("error unmarshal elliptic curve point")
	}

	p.x.Set(x)
	p.y.Set(y)
	return nil
}

// Add adds two points on the same curve
func (p *Point) Add(q *Point) *Point {
	x, y := curve.Add(p.x, p.y, q.x, q.y)
	return &Point{x: x, y: y}
}

// ScalarMult multiplies a point with a scalar
func (p *Point) ScalarMult(scalar []byte) *Point {
	x, y := curve.ScalarMult(p.x, p.y, scalar)
	return &Point{x: x, y: y}
}

// Sub substract point p with q
func (p *Point) Sub(q *Point) *Point {
	// p - q = p.x + q.x, p.y - q.y
	x, y := curve.Add(p.x, p.y, q.x, new(big.Int).Neg(q.y))
	return &Point{x: x, y: y}
}

// DeriveKeyFromECPoint returns a key of 32 byte from an elliptic curve point
func (p *Point) DeriveKeyFromECPoint() []byte {
	key := blake3.Sum256(p.x.Bytes())
	return key[:]
}

// GenerateKeyWithPoint returns a secret and public key pair
func GenerateKeyWithPoints() ([]byte, *Point, error) {
	secret, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, &Point{}, err
	}

	return secret, &Point{x: x, y: y}, nil
}

// pointWriter for elliptic curve points
type pointWriter struct {
	w io.Writer
}

// pointReader for elliptic curve points
type pointReader struct {
	r         io.Reader
	encodeLen int
}

// NewECPointWriter returns an elliptic curve point writer
func NewECPointWriter(w io.Writer) *pointWriter {
	return &pointWriter{w: w}
}

// NewECPointReader returns an elliptic curve point reader
func NewECPointReader(r io.Reader) *pointReader {
	return &pointReader{r: r, encodeLen: encodeLen}
}

// Write writes the marshalled elliptic curve point to writer
func (w *pointWriter) Write(p *Point) (err error) {
	_, err = w.w.Write(p.Marshal())
	return err
}

// Read reads a marshalled elliptic curve point from reader and stores it in point
func (r *pointReader) Read(p *Point) (err error) {
	pt := make([]byte, r.encodeLen)
	if _, err = io.ReadFull(r.r, pt); err != nil {
		return err
	}

	return p.Unmarshal(pt)
}
