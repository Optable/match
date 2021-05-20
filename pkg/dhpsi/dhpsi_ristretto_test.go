package dhpsi

import (
	"crypto/sha512"
	"testing"

	"github.com/bwesterb/go-ristretto"
	r255 "github.com/gtank/ristretto255"
)

var xxx = []byte("e:person@organization.tld")

func TestInterOperability(t *testing.T) {
	// gr
	var p1 ristretto.Point
	// derive
	p1.DeriveDalek(xxx)
	// print to hex
	var out1 [32]byte
	p1.BytesInto(&out1)

	// r255
	var p2 = r255.NewElement()
	// derive
	hash := sha512.Sum512(xxx)
	p2 = p2.FromUniformBytes(hash[:])
	// print to hex
	var tmp []byte
	tmp = p2.Encode(tmp)
	var out2 [32]byte
	copy(out2[:], tmp)

	if out1 != out2 {
		t.Fatal("go-ristretto and ristretto255 not producing the same output")
	}
}

func BenchmarkGRDeriveMultiply(b *testing.B) {
	// get a gr
	gr, _ := NewRistretto(RistrettoTypeGR)

	for i := 0; i < b.N; i++ {
		gr.DeriveMultiply([]byte(xxx))
	}
}

func BenchmarkGRMultiply(b *testing.B) {
	// get a gr
	gr, _ := NewRistretto(RistrettoTypeGR)
	m := gr.DeriveMultiply(xxx)

	for i := 0; i < b.N; i++ {
		gr.Multiply(m)
	}
}

func BenchmarkR255DeriveMultiply(b *testing.B) {
	// get a r255
	r255, _ := NewRistretto(RistrettoTypeR255)

	for i := 0; i < b.N; i++ {
		r255.DeriveMultiply([]byte(xxx))
	}
}

func Benchmark255Multiply(b *testing.B) {
	// get a r255
	r255, _ := NewRistretto(RistrettoTypeR255)
	m := r255.DeriveMultiply(xxx)

	for i := 0; i < b.N; i++ {
		r255.Multiply(m)
	}
}
