package util

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

var prng = rand.New(rand.NewSource(time.Now().UnixNano()))

func TestTranspose3D(t *testing.T) {
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([][][]byte, 4)
	for m := range b {
		b[m] = make([][]byte, 2)
		b[m][0] = make([]byte, 8)
		b[m][1] = make([]byte, 8)
		SampleBitSlice(prng, b[m][0])
		SampleBitSlice(prng, b[m][1])
	}

	for m := range b {
		if !bytes.Equal(b[m][0], Transpose3D(Transpose3D(b))[m][0]) {
			t.Fatalf("Transpose of transpose should be equal")
		}

		if !bytes.Equal(b[m][1], Transpose3D(Transpose3D(b))[m][1]) {
			t.Fatalf("Transpose of transpose should be equal")
		}
	}
}

func TestTranspose(t *testing.T) {

	b := make([][]byte, 4)
	for m := range b {
		b[m] = make([]byte, 8)
		SampleBitSlice(prng, b[m])
	}

	for m := range b {
		if !bytes.Equal(b[m], Transpose(Transpose(b))[m]) {
			t.Fatalf("Transpose of transpose should be equal")
		}
	}
}

func BenchmarkSampleBitMatrix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SampleRandomBitMatrix(prng, 10000, 424)
	}
}

func BenchmarkXorBytes(b *testing.B) {
	a := make([]byte, 10000)
	SampleBitSlice(prng, a)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		XorBytes(a, a)
	}
}

func BenchmarkInplaceXorBytes(b *testing.B) {
	a := make([]byte, 10000)
	SampleBitSlice(prng, a)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InPlaceXorBytes(a, a)
	}
}
