package ot

import (
	"crypto/elliptic"
	"crypto/rand"
	"math/big"

	"github.com/zeebo/blake3"
)

/*
High level api for operating on elliptic curve points.
*/

type points struct {
	curve elliptic.Curve
	x     *big.Int
	y     *big.Int
}

func newPoints(curve elliptic.Curve, x, y *big.Int) points {
	return points{curve: curve, x: x, y: y}
}

func (p points) add(q points) points {
	x, y := p.curve.Add(p.x, p.y, q.x, q.y)
	return newPoints(p.curve, x, y)
}

func (p points) scalarMult(scalar []byte) points {
	x, y := p.curve.ScalarMult(p.x, p.y, scalar)
	return newPoints(p.curve, x, y)
}

func (p points) sub(q points) points {
	// p - q = p.x + q.x, p.y - q.y
	n := big.NewInt(0)
	n.Neg(q.y)
	x, y := p.curve.Add(p.x, p.y, q.x, n)
	return newPoints(p.curve, x, y)
}

func (p points) isOnCurve() bool {
	return p.curve.IsOnCurve(p.x, p.y)
}

// deriveKey returns a key of 32 byte from an elliptic curve point
func (p points) deriveKey() []byte {
	buf := elliptic.Marshal(p.curve, p.x, p.y)
	key := blake3.Sum256(buf)
	return key[:]
}

func generateKeyWithPoints(curve elliptic.Curve) ([]byte, points, error) {
	secret, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, points{}, err
	}

	return secret, newPoints(curve, x, y), nil
}

// deriveKey returns a key of 32 byte from an elliptic curve point
func deriveKey(point []byte) []byte {
	key := blake3.Sum256(point)
	return key[:]
}
