package util

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/alecthomas/unsafeslice"
)

var prng = rand.New(rand.NewSource(time.Now().UnixNano()))

func sampleByteSlice(prng *rand.Rand, b []byte) (err error) {
	if _, err = prng.Read(b); err != nil {
		return nil
	}
	return nil
}

func sampleUint64Slice(prng *rand.Rand, u []uint64) {
	for i := range u {
		u[i] = prng.Uint64()
	}
}

func TestTestBitSetInByte(t *testing.T) {
	b := []byte{1}

	for i := 0; i < 8; i++ {
		if i == 0 {
			if TestBitSetInByte(b, i) != 1 {
				t.Fatalf("bit extraction failed")
			}
		} else {
			if TestBitSetInByte(b, i) != 0 {
				t.Fatalf("bit extraction failed")
			}
		}
	}

	b = []byte{161}
	for i := 0; i < 8; i++ {
		if i == 0 || i == 7 || i == 5 {
			if TestBitSetInByte(b, i) != 1 {
				t.Fatalf("bit extraction failed")
			}
		} else {
			if TestBitSetInByte(b, i) != 0 {
				t.Fatalf("bit extraction failed")
			}
		}

	}
}

// Note the double conversion of bytes to uint64s to bytes does
// result in added 0s.
func TestSliceConversions(t *testing.T) {
	lengths := []int{8, 16, 24, 32, 40, 48}
	for _, l := range lengths {
		// Bytes to Uint64s
		b := make([]byte, l)
		sampleByteSlice(prng, b)
		u := unsafeslice.Uint64SliceFromByteSlice(b)
		bb := unsafeslice.ByteSliceFromUint64Slice(u)

		// test
		for i, e := range b {
			if e != bb[i] {
				t.Errorf("Byte-to-Uint64-to-Byte conversion did not result in identical slices")
			}
		}
	}
	lengths = []int{2, 8, 16, 34, 100}
	for _, l := range lengths {
		// Uint64s to Bytes
		u := make([]uint64, l)
		sampleUint64Slice(prng, u)
		b := unsafeslice.ByteSliceFromUint64Slice(u)
		uu := unsafeslice.Uint64SliceFromByteSlice(b)

		//test
		for i, e := range u {
			if e != uu[i] {
				t.Errorf("Uint64-to-Byte-to-Uint64 conversion did not result in identical slices")
			}
		}

	}
}

func TestNaiveTranspose(t *testing.T) {
	b := make([][]byte, 4)
	for m := range b {
		b[m] = make([]byte, 8)
		prng.Read(b[m])
	}

	for m := range b {
		if !bytes.Equal(b[m], Transpose(Transpose(b))[m]) {
			t.Fatalf("Transpose of transpose should be equal")
		}
	}
}

func TestConcurrentInPlaceXorBytes(t *testing.T) {
	for _, l := range []int{10, 16384, 10000000} {
		// XOR with itself
		a := make([]byte, l)
		if _, err := prng.Read(a); err != nil {
			t.Fatalf("error generating random bytes")
		}
		ConcurrentInPlaceXorBytes(a, a)
		for _, i := range a {
			if i != 0 {
				t.Fatalf("XOR operation was not performed correctly")
			}
		}
		// doubly XOR with another slice to get back original
		c := make([]byte, l)
		d := make([]byte, l)
		e := make([]byte, l)
		if _, err := prng.Read(c); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(e); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(d, c) // save original to check later
		ConcurrentInPlaceXorBytes(c, e)
		ConcurrentInPlaceXorBytes(c, e)
		for i := range c {
			if c[i] != d[i] {
				t.Fatalf("performing concurrent XOR operation twice did not result in same slice")
			}
		}
		// XOR same original slice with concurrent and non-concurrent versions and then compare output
		f := make([]byte, l)
		g := make([]byte, l)
		h := make([]byte, l)
		if _, err := prng.Read(f); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(h); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(g, f)
		ConcurrentInPlaceXorBytes(f, h)
		InPlaceXorBytes(g, h)
		for i := range f {
			if f[i] != g[i] {
				t.Fatalf("result of concurrent XOR operation did not match with result of non-concurrent equivalent")
			}
		}
	}
}

func TestConcurrentUnsafeInPlaceXorBytes(t *testing.T) {
	for _, l := range []int{10, 16384, 10000000} {
		// XOR with itself
		a := make([]byte, l)
		if _, err := prng.Read(a); err != nil {
			t.Fatalf("error generating random bytes")
		}
		ConcurrentUnsafeInPlaceXorBytes(a, a)
		for _, i := range a {
			if i != 0 {
				t.Fatalf("XOR operation was not performed correctly")
			}
		}
		// doubly XOR with another slice to get back original
		c := make([]byte, l)
		d := make([]byte, l)
		e := make([]byte, l)
		if _, err := prng.Read(c); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(e); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(d, c) // save original to check later
		ConcurrentUnsafeInPlaceXorBytes(c, e)
		ConcurrentUnsafeInPlaceXorBytes(c, e)
		for i := range c {
			if c[i] != d[i] {
				t.Fatalf("performing concurrent XOR operation twice did not result in same slice")
			}
		}
		// XOR same original slice with concurrent and non-concurrent versions and then compare output
		f := make([]byte, l)
		g := make([]byte, l)
		h := make([]byte, l)
		if _, err := prng.Read(f); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(h); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(g, f)
		ConcurrentUnsafeInPlaceXorBytes(f, h)
		InPlaceXorBytes(g, h)
		for i := range f {
			if f[i] != g[i] {
				t.Fatalf("result of concurrent XOR operation did not match with result of non-concurrent equivalent")
			}
		}
	}
}

func TestUnsafeInPlaceXorBytes(t *testing.T) {
	for _, l := range []int{10, 16384, 10000000} {
		// XOR with itself
		a := make([]byte, l)
		if _, err := prng.Read(a); err != nil {
			t.Fatalf("error generating random bytes")
		}
		xor(a, a)
		for _, i := range a {
			if i != 0 {
				t.Fatalf("XOR operation was not performed correctly")
			}
		}
		// doubly XOR with another slice to get back original
		c := make([]byte, l)
		d := make([]byte, l)
		e := make([]byte, l)
		if _, err := prng.Read(c); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(e); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(d, c) // save original to check later
		xor(c, e)
		xor(c, e)
		for i := range c {
			if c[i] != d[i] {
				t.Fatalf("performing concurrent XOR operation twice did not result in same slice")
			}
		}
		// XOR same original slice with concurrent and non-concurrent versions and then compare output
		f := make([]byte, l)
		g := make([]byte, l)
		h := make([]byte, l)
		if _, err := prng.Read(f); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(h); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(g, f)
		xor(f, h)
		InPlaceXorBytes(g, h)
		for i := range f {
			if f[i] != g[i] {
				t.Fatalf("result of concurrent XOR operation did not match with result of non-concurrent equivalent")
			}
		}
	}
}

func TestConcurrentInPlaceAndBytes(t *testing.T) {
	for _, l := range []int{10, 16384, 10000000} {
		// XOR with itself
		a := make([]byte, l)
		b := make([]byte, l)
		if _, err := prng.Read(a); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(b, a)
		ConcurrentInPlaceAndBytes(a, a)
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("AND operation was not performed correctly")
			}
		}
		// AND same original slice with concurrent and non-concurrent versions and then compare output
		f := make([]byte, l)
		g := make([]byte, l)
		h := make([]byte, l)
		if _, err := prng.Read(f); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(h); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(g, f)
		ConcurrentInPlaceAndBytes(f, h)
		InPlaceAndBytes(g, h)
		for i := range f {
			if f[i] != g[i] {
				t.Fatalf("result of concurrent AND operation did not match with result of non-concurrent equivalent")
			}
		}
	}

}

func BenchmarkXorBytes(b *testing.B) {
	a := make([]byte, 10000)
	prng.Read(a)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		XorBytes(a, a)
	}
}

func BenchmarkInPlaceXorBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InPlaceXorBytes(a, a)
	}
}

func BenchmarkUnsafeInPlaceXorBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xor(a, a)
	}
}

func BenchmarkConcurrentInPlaceXorBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentInPlaceXorBytes(a, a)
	}
}

func BenchmarkConcurrentUnsafeInPlaceXorBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentUnsafeInPlaceXorBytes(a, a)
	}
}

func BenchmarkConcurrentUnsafeBitOperation(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentBitOp(xor, a, a)
	}
}

func BenchmarkInPlaceAndBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InPlaceAndBytes(a, a)
	}
}

func BenchmarkConcurrentInPlaceAndBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentInPlaceAndBytes(a, a)
	}
}
