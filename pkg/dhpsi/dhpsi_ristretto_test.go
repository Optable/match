package dhpsi

import (
	"crypto/sha512"
	"testing"

	"github.com/bwesterb/go-ristretto"
	r255 "github.com/gtank/ristretto255"
)

var xxx = []byte("e:person@organization.tld")

type NilRistretto int

// test loopback ristretto just copies data out
// and does no treatment
func (g NilRistretto) DeriveMultiply(dst *[EncodedLen]byte, src []byte) {
	// return first 32 bytes of matchable
	copy(dst[:], src[:32])
}
func (g NilRistretto) Multiply(dst *[EncodedLen]byte, src [EncodedLen]byte) {
	// passthrought
	copy(dst[:], src[:])
}

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

func TestInterInterfaceOperability(t *testing.T) {
	// gr
	gr, _ := NewRistretto(RistrettoTypeGR)
	var point1 [EncodedLen]byte
	gr.DeriveMultiply(&point1, xxx)

	// r255
	r255, _ := NewRistretto(RistrettoTypeR255)
	var point2 [EncodedLen]byte
	r255.DeriveMultiply(&point2, xxx)

	if point1 != point1 {
		t.Fatal("RistrettoTypeGR and RistrettoTypeR255 DeriveMultiply not producing the same output")
	}

	// todo: this second test needs the same key set on both ristretto implementation
	/*
		// gr
		var point3 [EncodedLen]byte
		gr.Multiply(&point3, point1)

		// r255
		var point4 [EncodedLen]byte
		r255.Multiply(&point4, point2)

		if point3 != point4 {
			t.Fatalf("RistrettoTypeGR and RistrettoTypeR255 Multiply not producing the same output (%s : %s", string(point3[:]), string(point4[:]))
		}
	*/

}

func BenchmarkGRDeriveMultiply(b *testing.B) {
	// get a gr
	gr, _ := NewRistretto(RistrettoTypeGR)
	var point [EncodedLen]byte

	for i := 0; i < b.N; i++ {
		gr.DeriveMultiply(&point, xxx)
	}
}

func BenchmarkGRMultiply(b *testing.B) {
	// get a gr
	gr, _ := NewRistretto(RistrettoTypeGR)
	var src [EncodedLen]byte
	gr.DeriveMultiply(&src, xxx)
	var dst [EncodedLen]byte

	for i := 0; i < b.N; i++ {
		gr.Multiply(&dst, src)
	}
}

func BenchmarkR255DeriveMultiply(b *testing.B) {
	// get a r255
	r255, _ := NewRistretto(RistrettoTypeR255)
	var point [EncodedLen]byte

	for i := 0; i < b.N; i++ {
		r255.DeriveMultiply(&point, xxx)
	}
}

func Benchmark255Multiply(b *testing.B) {
	// get a r255
	r255, _ := NewRistretto(RistrettoTypeR255)
	var src [EncodedLen]byte
	r255.DeriveMultiply(&src, xxx)
	var dst [EncodedLen]byte

	for i := 0; i < b.N; i++ {
		r255.Multiply(&dst, src)
	}
}
