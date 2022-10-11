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

// encodeLenWithCurve returns the number of bytes needed to encode a point
func encodeLenWithCurve(curve elliptic.Curve) int {
	return len(elliptic.MarshalCompressed(curve, curve.Params().Gx, curve.Params().Gy))
}

// Point represents a point on the P256 elliptic curve
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
		return fmt.Errorf("error unmarshalling elliptic curve point")
	}

	p.x.Set(x)
	p.y.Set(y)
	return nil
}

// Add adds two points
func (p *Point) Add(q *Point) *Point {
	x, y := curve.Add(p.x, p.y, q.x, q.y)
	return &Point{x: x, y: y}
}

// ScalarMult multiplies a point with a scalar
func (p *Point) ScalarMult(scalar []byte) *Point {
	x, y := curve.ScalarMult(p.x, p.y, scalar)
	return &Point{x: x, y: y}
}

// Sub substracts point p from q
func (p *Point) Sub(q *Point) *Point {
	// in order to do point subtraction, we need to make sure
	// the negative point is still mapped properly in the field elements.
	negQy := new(big.Int).Neg(q.y)
	negQy = negQy.Mod(negQy, curve.Params().P) // here P is the order of the curve field

	// p - q = p.x + q.x, p.y - q.y
	x, y := curve.Add(p.x, p.y, q.x, negQy)
	return &Point{x: x, y: y}
}

// DeriveKeyFromECPoint returns a key of 32 byte
func (p *Point) DeriveKeyFromECPoint() []byte {
	key := blake3.Sum256(p.x.Bytes())
	return key[:]
}

// GenerateKey returns a secret and public key pair
func GenerateKey() ([]byte, *Point, error) {
	secret, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	return secret, &Point{x: x, y: y}, nil
}

// pointWriter for elliptic curve points
type pointWriter struct {
	w io.Writer
}

// pointReader for elliptic curve points
type pointReader struct {
	r io.Reader
}

// NewECPointWriter returns an elliptic curve point writer
func NewECPointWriter(w io.Writer) *pointWriter {
	return &pointWriter{w: w}
}

// NewECPointReader returns an elliptic curve point reader
func NewECPointReader(r io.Reader) *pointReader {
	return &pointReader{r: r}
}

// Write writes the marshalled elliptic curve point to writer
func (w *pointWriter) Write(p *Point) (err error) {
	_, err = w.w.Write(p.Marshal())
	return err
}

// Read reads a marshalled elliptic curve point from reader and stores it in point
func (r *pointReader) Read(p *Point) (err error) {
	pt := make([]byte, encodeLen)
	if _, err = io.ReadFull(r.r, pt); err != nil {
		return err
	}

	return p.Unmarshal(pt)
}

// Equal returns true when 2 points are equal
func (p *Point) equal(q *Point) bool {
	return p.x.Cmp(q.x) == 0 && p.y.Cmp(q.y) == 0
}
