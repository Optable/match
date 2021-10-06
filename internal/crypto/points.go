package crypto

import (
	"crypto/elliptic"
	"crypto/rand"
	"math/big"

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

type Points struct {
	curve elliptic.Curve
	x     *big.Int
	y     *big.Int
}

func NewPoints(curve elliptic.Curve, x, y *big.Int) Points {
	return Points{curve: curve, x: x, y: y}
}

func (p Points) SetX(newX *big.Int) Points {
	p.x.Set(newX)
	return p
}

func (p Points) SetY(newY *big.Int) Points {
	p.y.Set(newY)
	return p
}

func (p Points) Marshal(curve elliptic.Curve) []byte {
	return elliptic.Marshal(curve, p.x, p.y)
}

func (p Points) Add(q Points) Points {
	x, y := p.curve.Add(p.x, p.y, q.x, q.y)
	return NewPoints(p.curve, x, y)
}

func (p Points) ScalarMult(scalar []byte) Points {
	x, y := p.curve.ScalarMult(p.x, p.y, scalar)
	return NewPoints(p.curve, x, y)
}

func (p Points) Sub(q Points) Points {
	// p - q = p.x + q.x, p.y - q.y
	x := big.NewInt(0)
	x, y := p.curve.Add(p.x, p.y, q.x, x.Neg(q.y))
	return NewPoints(p.curve, x, y)
}

func (p Points) IsOnCurve() bool {
	return p.curve.IsOnCurve(p.x, p.y)
}

// deriveKey returns a key of 32 byte from an elliptic curve point
func (p Points) DeriveKey() []byte {
	key := blake3.Sum256(p.x.Bytes())
	return key[:]
}

func GenerateKeyWithPoints(curve elliptic.Curve) ([]byte, Points, error) {
	secret, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, Points{}, err
	}

	return secret, NewPoints(curve, x, y), nil
}

// deriveKey returns a key of 32 byte from an elliptic curve point
func DeriveKey(point []byte) []byte {
	key := blake3.Sum256(point)
	return key[:]
}
