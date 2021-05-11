package dhpsi

import "testing"

var xxx = []byte("e:person@organization.tld")

func BenchmarkGRDeriveMultiply(b *testing.B) {
	// get a gr
	gr := NewRistretto(RistrettoTypeGR)

	for i := 0; i < b.N; i++ {
		gr.DeriveMultiply([]byte(xxx))
	}
}

func BenchmarkGRMultiply(b *testing.B) {
	// get a gr
	gr := NewRistretto(RistrettoTypeGR)
	m := gr.DeriveMultiply(xxx)

	for i := 0; i < b.N; i++ {
		gr.Multiply(m)
	}
}

func BenchmarkR255DeriveMultiply(b *testing.B) {
	// get a r255
	r255 := NewRistretto(RistrettoTypeR255)

	for i := 0; i < b.N; i++ {
		r255.DeriveMultiply([]byte(xxx))
	}
}

func Benchmark255Multiply(b *testing.B) {
	// get a r255
	r255 := NewRistretto(RistrettoTypeR255)
	m := r255.DeriveMultiply(xxx)

	for i := 0; i < b.N; i++ {
		r255.Multiply(m)
	}
}
