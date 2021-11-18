package crypto

import (
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/zeebo/blake3"
)

var (
	c  = elliptic.P256()
	bx *big.Int
	by *big.Int
)

func arePointsEqual(p Point, q Point) bool {
	return p.x.Cmp(q.x) == 0 && p.y.Cmp(q.y) == 0 && p.curve.Params().Name == q.curve.Params().Name
}

func TestNewPoint(t *testing.T) {
	x := big.NewInt(1)
	y := big.NewInt(2)
	point := Point{curve: c, x: x, y: y}
	n := point.curve.Params().Name
	if n != "P-256" {
		t.Fatalf("newPoints curve: want :P-256, got %s", n)
	}
	if x.Cmp(point.x) != 0 {
		t.Fatalf("newPoints x: want %d, got %d", x.Int64(), point.x.Int64())
	}
	if y.Cmp(point.y) != 0 {
		t.Fatalf("newPoints y: want %d, got %d", y.Int64(), point.y.Int64())
	}
}

func TestAdd(t *testing.T) {
	bx, by = c.Params().Gx, c.Params().Gy
	p := Point{curve: c, x: bx, y: by}
	p = p.Add(p)

	dx, dy := c.Double(bx, by)
	if !arePointsEqual(p, Point{curve: c, x: dx, y: dy}) {
		t.Fatal("Error in points addition.")
	}
}

func TestScalarMult(t *testing.T) {
	a, dx, dy, _ := elliptic.GenerateKey(c, rand.Reader)
	p := Point{curve: c, x: bx, y: by}

	dp := p.ScalarMult(a)

	if !arePointsEqual(Point{curve: c, x: dx, y: dy}, dp) {
		t.Fatal("Error in points scalar multiplication.")
	}
}

func TestSub(t *testing.T) {
	_, dx, dy, _ := elliptic.GenerateKey(c, rand.Reader)
	p := Point{curve: c, x: dx, y: dy}
	s := p.Add(p)
	s = s.Sub(p)

	if s.x.Cmp(p.x) != 0 || s.y.Cmp(p.y) != 0 {
		t.Fatalf("Error in points substraction. want: %v, %v, got %v, %v", p.x, p.y, s.x, s.y)
	}
}

func TestDeriveKeyPoint(t *testing.T) {
	p := Point{curve: c, x: bx, y: by}
	key := p.DeriveKeyFromECPoint()

	key2 := blake3.Sum256(bx.Bytes())
	if string(key) != string(key2[:]) || len(key) != 32 {
		t.Fatal("Error in points deriveKey")
	}
}

func TestGenerateKeyWithPoint(t *testing.T) {
	p := Point{curve: c, x: bx, y: by}
	s, P, err := GenerateKeyWithPoints(c)
	if err != nil {
		t.Fatal(err)
	}

	d := p.ScalarMult(s)
	if !arePointsEqual(d, P) {
		t.Fatal("Error in points generateKeyWithPoints")
	}
}

func BenchmarkDeriveKey(b *testing.B) {
	x := big.NewInt(1)
	y := big.NewInt(2)
	p := Point{curve: c, x: x, y: y}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.DeriveKeyFromECPoint()
	}
}

func BenchmarkSub(b *testing.B) {
	x := big.NewInt(1)
	y := big.NewInt(2)
	p := Point{curve: c, x: x, y: y}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Sub(p)
	}
}
