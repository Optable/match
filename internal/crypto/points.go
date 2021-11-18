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
High level api for operating on elliptic curve Points.
*/

// LenEncodeOnCurve returns the number of bytes needed to encode a point
// on the curve
func LenEncodeOnCurve(curve elliptic.Curve) int {
	return len(elliptic.Marshal(curve, curve.Params().Gx, curve.Params().Gy))
}

// Points represents a point on an elliptic curve
type Points struct {
	curve elliptic.Curve
	x     *big.Int
	y     *big.Int
}

// NewPoints returns a blank points on an elliptic curve
func NewPoints(curve elliptic.Curve) Points {
	return Points{curve: curve, x: new(big.Int), y: new(big.Int)}
}

func newPoints(curve elliptic.Curve, x, y *big.Int) Points {
	return Points{curve: curve, x: x, y: y}
}

// SetX sets the x coordiante of a point on elliptic curve
func (p Points) SetX(newX *big.Int) Points {
	p.x.Set(newX)
	return p
}

// SetY sets the y coordinate of a point on an elliptic curve
func (p Points) SetY(newY *big.Int) Points {
	p.y.Set(newY)
	return p
}

// Marshal converts a Points to a byte slice representation
func (p Points) Marshal() []byte {
	return elliptic.Marshal(p.curve, p.x, p.y)
}

// Unmarshal takes in a marshaledPoint byte slice and extracts the Points object
func (p Points) Unmarshal(marshaledPoint []byte) error {
	x, y := elliptic.Unmarshal(p.curve, marshaledPoint)

	// on error of Unmarshal, x is nil
	if x == nil {
		return fmt.Errorf("error unmarshal elliptic curve point")
	}

	p.x.Set(x)
	p.y.Set(y)
	return nil
}

// Add adds two points on the same curve
func (p Points) Add(q Points) Points {
	x, y := p.curve.Add(p.x, p.y, q.x, q.y)
	return newPoints(p.curve, x, y)
}

// ScalarMult multiplies a points with a scalar
func (p Points) ScalarMult(scalar []byte) Points {
	x, y := p.curve.ScalarMult(p.x, p.y, scalar)
	return newPoints(p.curve, x, y)
}

// Sub substract points p with q
func (p Points) Sub(q Points) Points {
	// p - q = p.x + q.x, p.y - q.y
	x := big.NewInt(0)
	x, y := p.curve.Add(p.x, p.y, q.x, x.Neg(q.y))
	return newPoints(p.curve, x, y)
}

// DeriveKeyFromECPoints returns a key of 32 byte from an elliptic curve point
func (p Points) DeriveKeyFromECPoints() []byte {
	key := blake3.Sum256(p.x.Bytes())
	return key[:]
}

// GenerateKeyWithPoints returns a secret and public key pair
func GenerateKeyWithPoints(curve elliptic.Curve) ([]byte, Points, error) {
	secret, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, Points{}, err
	}

	return secret, newPoints(curve, x, y), nil
}

// pointsWriter for elliptic curve points
type pointsWriter struct {
	w io.Writer
}

// pointsReader for elliptic curve points
type pointsReader struct {
	r         io.Reader
	encodeLen int
}

// NewECPointsWriter returns an elliptic curve point writer
func NewECPointsWriter(w io.Writer) *pointsWriter {
	return &pointsWriter{w: w}
}

// NewECPointsReader returns an elliptic curve point reader
func NewECPointsReader(r io.Reader, l int) *pointsReader {
	return &pointsReader{r: r, encodeLen: l}
}

// Write writes the marshalled elliptic curve point to writer
func (w *pointsWriter) Write(p Points) (err error) {
	_, err = w.w.Write(p.Marshal())
	return err
}

// Read reads a marshalled elliptic curve point from reader and stores it in point
func (r *pointsReader) Read(p Points) (err error) {
	pt := make([]byte, r.encodeLen)
	if _, err = io.ReadFull(r.r, pt); err != nil {
		return err
	}

	return p.Unmarshal(pt)
}
