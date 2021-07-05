package ot

import (
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/zeebo/blake3"
)

var (
	c  elliptic.Curve
	bx *big.Int
	by *big.Int
)

func arePointsEqual(p points, q points) bool {
	return p.x.Cmp(q.x) == 0 && p.y.Cmp(q.y) == 0 && p.curve.Params().Name == q.curve.Params().Name
}

func TestNewPoints(t *testing.T) {
	c, _ = initCurve(curve)
	x := big.NewInt(1)
	y := big.NewInt(2)

	points := newPoints(c, x, y)
	n := points.curve.Params().Name
	if n != "P-256" {
		t.Fatalf("newPoints curve: want :P-256, got %s", n)
	}
	if x.Cmp(points.x) != 0 {
		t.Fatalf("newPoints x: want %d, got %d", x.Int64(), points.x.Int64())
	}
	if y.Cmp(points.y) != 0 {
		t.Fatalf("newPoints y: want %d, got %d", y.Int64(), points.y.Int64())
	}
}

func TestAdd(t *testing.T) {
	bx, by = c.Params().Gx, c.Params().Gy
	p := newPoints(c, bx, by)
	p = p.add(p)

	dx, dy := c.Double(bx, by)
	if !arePointsEqual(p, newPoints(c, dx, dy)) {
		t.Fatal("Error in points addition.")
	}
}

func TestScalarMult(t *testing.T) {
	a, dx, dy, _ := elliptic.GenerateKey(c, rand.Reader)
	p := newPoints(c, bx, by)

	dp := p.scalarMult(a)

	if !arePointsEqual(newPoints(c, dx, dy), dp) {
		t.Fatal("Error in points scalar multiplication.")
	}
}

func TestSub(t *testing.T) {
	_, dx, dy, _ := elliptic.GenerateKey(c, rand.Reader)
	p := newPoints(c, dx, dy)
	s := p.add(p)
	s = s.sub(p)

	if s.x.Cmp(p.x) != 0 || s.y.Cmp(p.y) != 0 {
		t.Fatalf("Error in points substraction. want: %v, %v, got %v, %v", p.x, p.y, s.x, s.y)
	}
}

func TestIsOnCurve(t *testing.T) {
	newC, _ := initCurve("P521")
	q := newPoints(c, newC.Params().Gx, newC.Params().Gy)
	r := newPoints(newC, newC.Params().Gx, newC.Params().Gy)

	if q.isOnCurve() || !r.isOnCurve() {
		t.Fatal("Error in points isOnCurve")
	}
}

func TestDeriveKeyPoints(t *testing.T) {
	p := newPoints(c, bx, by)
	key := p.deriveKey()

	key2 := blake3.Sum256(elliptic.Marshal(c, bx, by))
	if string(key) != string(key2[:]) || len(key) != 32 {
		t.Fatal("Error in points deriveKey")
	}
}

func TestGenerateKeyWithPoints(t *testing.T) {
	p := newPoints(c, bx, by)
	s, P, err := generateKeyWithPoints(c)
	if err != nil {
		t.Fatal(err)
	}

	d := p.scalarMult(s)
	if !arePointsEqual(d, P) {
		t.Fatal("Error in points generateKeyWithPoints")
	}
}
