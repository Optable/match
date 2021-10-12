package crypto

import (
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"

	gr "github.com/bwesterb/go-ristretto"
	"github.com/zeebo/blake3"
)

/*
High level api for operating on elliptic curve Points.
*/

const (
	P224 = "P224"
	P256 = "P256"
	P384 = "P384"
	P521 = "P521"
)

// InitCurve instantiate an elliptic curve object given the curveName and returns the number of bytes
// needed to encode a point on the curve
func InitCurve(curveName string) (curve elliptic.Curve, encodeLen int) {
	switch curveName {
	case P224:
		curve = elliptic.P224()
	case P256:
		curve = elliptic.P256()
	case P384:
		curve = elliptic.P384()
	case P521:
		curve = elliptic.P521()
	default:
		curve = elliptic.P256()
	}
	encodeLen = len(elliptic.Marshal(curve, curve.Params().Gx, curve.Params().Gy))
	return
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

// SetY sets the y coordiante of a point on elliptic curve
func (p Points) SetY(newY *big.Int) Points {
	p.y.Set(newY)
	return p
}

// Marshal converts a points to a byte slice representation
func (p Points) Marshal() []byte {
	return elliptic.Marshal(p.curve, p.x, p.y)
}

// Unmarshal takes in a marshaledPoint byte slice and extracts the points object
func (p Points) Unmarshal(marshaledPoint []byte) error {
	x, y := elliptic.Unmarshal(p.curve, marshaledPoint)

	// on error of unmarshal, x is nil
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

// DeriveKey returns a key of 32 byte from an elliptic curve point
func (p Points) DeriveKey() []byte {
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

// DeriveKey returns a key of 32 byte from an elliptic curve point
func deriveKey(point []byte) []byte {
	key := blake3.Sum256(point)
	return key[:]
}

// deriveKey returns a key of 32 byte from an elliptic curve point
func DeriveKeyRistretto(point *gr.Point) ([]byte, error) {
	buf, err := point.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return deriveKey(buf), nil
}
