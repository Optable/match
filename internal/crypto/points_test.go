package crypto

import (
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/zeebo/blake3"
)

var (
	c     elliptic.Curve
	bx    *big.Int
	by    *big.Int
	curve = "P256"
)

func TestInitCurve(t *testing.T) {
	curveTests := []struct {
		name string
		want string
	}{
		{"P224", "P-224"},
		{"P256", "P-256"},
		{"P384", "P-384"},
		{"P521", "P-521"},
	}

	for _, tt := range curveTests {
		c, _ := InitCurve(tt.name)
		got := c.Params().Name
		if got != tt.want {
			t.Fatalf("InitCurve(%s): want curve %s, got curve %s", tt.name, tt.name, got)
		}
	}
}

func arePointsEqual(p Points, q Points) bool {
	return p.x.Cmp(q.x) == 0 && p.y.Cmp(q.y) == 0 && p.curve.Params().Name == q.curve.Params().Name
}

func TestNewPoints(t *testing.T) {
	c, _ = InitCurve(curve)
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
	p = p.Add(p)

	dx, dy := c.Double(bx, by)
	if !arePointsEqual(p, newPoints(c, dx, dy)) {
		t.Fatal("Error in points addition.")
	}
}

func TestScalarMult(t *testing.T) {
	a, dx, dy, _ := elliptic.GenerateKey(c, rand.Reader)
	p := newPoints(c, bx, by)

	dp := p.ScalarMult(a)

	if !arePointsEqual(newPoints(c, dx, dy), dp) {
		t.Fatal("Error in points scalar multiplication.")
	}
}

func TestSub(t *testing.T) {
	_, dx, dy, _ := elliptic.GenerateKey(c, rand.Reader)
	p := newPoints(c, dx, dy)
	s := p.Add(p)
	s = s.Sub(p)

	if s.x.Cmp(p.x) != 0 || s.y.Cmp(p.y) != 0 {
		t.Fatalf("Error in points substraction. want: %v, %v, got %v, %v", p.x, p.y, s.x, s.y)
	}
}

func TestDeriveKeyPoints(t *testing.T) {
	p := newPoints(c, bx, by)
	key := p.DeriveKey()

	key2 := blake3.Sum256(bx.Bytes())
	if string(key) != string(key2[:]) || len(key) != 32 {
		t.Fatal("Error in points deriveKey")
	}
}

func TestGenerateKeyWithPoints(t *testing.T) {
	p := newPoints(c, bx, by)
	s, P, err := GenerateKeyWithPoints(c)
	if err != nil {
		t.Fatal(err)
	}

	d := p.ScalarMult(s)
	if !arePointsEqual(d, P) {
		t.Fatal("Error in points generateKeyWithPoints")
	}
}

func TestDeriveKey(t *testing.T) {
	c := elliptic.P256()
	_, px, py, err := elliptic.GenerateKey(c, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	p := elliptic.Marshal(c, px, py)
	key := DeriveKey(p)
	if len(key) != 32 {
		t.Fatalf("derived key length is not 32, got: %d", len(key))
	}
}

func BenchmarkDeriveKey(b *testing.B) {
	c, _ = InitCurve(curve)
	x := big.NewInt(1)
	y := big.NewInt(2)
	p := newPoints(c, x, y)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.DeriveKey()
	}
}

func BenchmarkSub(b *testing.B) {
	c, _ = InitCurve(curve)
	x := big.NewInt(1)
	y := big.NewInt(2)
	p := newPoints(c, x, y)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Sub(p)
	}
}
