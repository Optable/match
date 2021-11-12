package util

import (
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/alecthomas/unsafeslice"
)

var prng = rand.New(rand.NewSource(time.Now().UnixNano()))
var benchmarkBytes = 10000000

// Note the double conversion of bytes to uint64s to bytes does
// result in added 0s.
// Only tested on AMD64.
func TestSliceConversions(t *testing.T) {
	lengths := []int{8, 16, 24, 32, 40, 48}
	for _, l := range lengths {
		// Bytes to Uint64s
		b := make([]byte, l)
		if _, err := prng.Read(b); err != nil {
			t.Fatal("error generating random bytes")
		}
		u := unsafeslice.Uint64SliceFromByteSlice(b)
		bb := unsafeslice.ByteSliceFromUint64Slice(u)

		// test
		for i, e := range b {
			if e != bb[i] {
				t.Error("byte-to-uint64-to-byte conversion did not result in identical slices")
			}
		}
	}
	lengths = []int{2, 8, 16, 34, 100}
	for _, l := range lengths {
		// Uint64s to Bytes
		u := make([]uint64, l)
		for i := range u {
			u[i] = prng.Uint64()
		}
		b := unsafeslice.ByteSliceFromUint64Slice(u)
		uu := unsafeslice.Uint64SliceFromByteSlice(b)

		//test
		for i, e := range u {
			if e != uu[i] {
				t.Error("uint64-to-byte-to-uint64 conversion did not result in identical slices")
			}
		}

	}
}

func TestXor(t *testing.T) {
	lengths := []int{3, 8, 16, 33}
	for _, l := range lengths {
		src := make([]byte, l)
		if _, err := prng.Read(src); err != nil {
			t.Fatal("error generating random bytes")
		}

		dst := make([]byte, l)
		if _, err := prng.Read(dst); err != nil {
			t.Fatal("error generating random bytes")
		}

		// copy for testing later
		cop := make([]byte, l)
		copy(cop, dst)

		err := Xor(dst, src)
		if err != nil {
			t.Error("bitwise XOR operation failed")
		}

		for i := range src {
			if dst[i] != cop[i]^src[i] {
				t.Error("bitwise XOR operation was not performed properly")
			}
		}
	}
}

func TestAnd(t *testing.T) {
	lengths := []int{3, 8, 16, 33}
	for _, l := range lengths {
		src := make([]byte, l)
		if _, err := prng.Read(src); err != nil {
			t.Fatal("error generating random bytes")
		}

		dst := make([]byte, l)
		if _, err := prng.Read(dst); err != nil {
			t.Fatal("error generating random bytes")
		}

		// copy for testing later
		cop := make([]byte, l)
		copy(cop, dst)

		err := And(dst, src)
		if err != nil {
			t.Error("bitwise AND operation failed")
		}

		for i := range src {
			if dst[i] != cop[i]&src[i] {
				t.Error("bitwise AND operation was not performed properly")
			}
		}
	}
}

func TestDoubleXor(t *testing.T) {
	lengths := []int{3, 8, 16, 33}
	for _, l := range lengths {
		src := make([]byte, l)
		if _, err := prng.Read(src); err != nil {
			t.Fatal("error generating random bytes")
		}

		src2 := make([]byte, l)
		if _, err := prng.Read(src2); err != nil {
			t.Fatal("error generating random bytes")
		}

		dst := make([]byte, l)
		if _, err := prng.Read(dst); err != nil {
			t.Fatal("error generating random bytes")
		}

		// copy for testing later
		cop := make([]byte, l)
		copy(cop, dst)

		err := DoubleXor(dst, src, src2)
		if err != nil {
			t.Error("bitwise double XOR operation failed")
		}

		for i := range src {
			if dst[i] != cop[i]^src[i]^src2[i] {
				t.Error("bitwise double XOR operation was not performed properly")
			}
		}
	}
}

func TestAndXor(t *testing.T) {
	lengths := []int{3, 8, 16, 33}
	for _, l := range lengths {
		src := make([]byte, l)
		if _, err := prng.Read(src); err != nil {
			t.Fatal("error generating random bytes")
		}

		src2 := make([]byte, l)
		if _, err := prng.Read(src2); err != nil {
			t.Fatal("error generating random bytes")
		}

		dst := make([]byte, l)
		if _, err := prng.Read(dst); err != nil {
			t.Fatal("error generating random bytes")
		}

		// copy for testing later
		cop := make([]byte, l)
		copy(cop, dst)

		err := AndXor(dst, src, src2)
		if err != nil {
			t.Error("bitwise AND followed by bitwise XOR operation failed")
		}

		for i := range src {
			if dst[i] != cop[i]&src[i]^src2[i] {
				t.Error("bitwise AND followed by bitwise XOR operation was not performed properly")
			}
		}
	}
}

func TestConcurrentBitOp(t *testing.T) {
	lengths := []int{3, 16, 16384 * runtime.GOMAXPROCS(0), 2 * 16384 * runtime.GOMAXPROCS(0)}
	for _, l := range lengths {
		src := make([]byte, l)
		if _, err := prng.Read(src); err != nil {
			t.Fatal("error generating random bytes")
		}

		dst := make([]byte, l)
		if _, err := prng.Read(dst); err != nil {
			t.Fatal("error generating random bytes")
		}

		// copy for testing later
		cop := make([]byte, l)
		copy(cop, dst)

		err := ConcurrentBitOp(Xor, dst, src)
		if err != nil {
			t.Error("concurrent bitwise XOR operation failed")
		}

		for i := range src {
			if dst[i] != cop[i]^src[i] {
				t.Error("concurrent bitwise XOR operation was not performed properly")
			}
		}
	}
}

func TestConcurrentDoubleBitOp(t *testing.T) {
	lengths := []int{3, 16, 16384 * runtime.GOMAXPROCS(0), 2 * 16384 * runtime.GOMAXPROCS(0)}
	for _, l := range lengths {
		src := make([]byte, l)
		if _, err := prng.Read(src); err != nil {
			t.Fatal("error generating random bytes")
		}

		src2 := make([]byte, l)
		if _, err := prng.Read(src2); err != nil {
			t.Fatal("error generating random bytes")
		}

		dst := make([]byte, l)
		if _, err := prng.Read(dst); err != nil {
			t.Fatal("error generating random bytes")
		}

		// copy for testing later
		cop := make([]byte, l)
		copy(cop, dst)

		err := ConcurrentDoubleBitOp(AndXor, dst, src, src2)
		if err != nil {
			t.Error("concurrent bitwise AND followed by bitwise XOR operation failed")
		}

		for i := range src {
			if dst[i] != cop[i]&src[i]^src2[i] {
				t.Error("concurrent bitwise AND followed by bitwise XOR operation was not performed properly")
			}
		}
	}
}

func TestTestBitSetInByte(t *testing.T) {
	b := []byte{1}

	for i := 0; i < 8; i++ {
		if i == 0 {
			if !BitSetInByte(b, i) {
				t.Fatal("bit extraction failed")
			}
		} else {
			if BitSetInByte(b, i) {
				t.Fatal("bit extraction failed")
			}
		}
	}

	b = []byte{161}
	for i := 0; i < 8; i++ {
		if i == 0 || i == 7 || i == 5 {
			if !BitSetInByte(b, i) {
				t.Fatal("bit extraction failed")
			}
		} else {
			if BitSetInByte(b, i) {
				t.Fatal("bit extraction failed")
			}
		}

	}
}

func BenchmarkXor(b *testing.B) {
	src := make([]byte, benchmarkBytes)
	if _, err := prng.Read(src); err != nil {
		b.Fatal("error generating random bytes")
	}

	dst := make([]byte, benchmarkBytes)
	if _, err := prng.Read(dst); err != nil {
		b.Fatal("error generating random bytes")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Xor(dst, src)
	}
}

func BenchmarkAnd(b *testing.B) {
	src := make([]byte, benchmarkBytes)
	if _, err := prng.Read(src); err != nil {
		b.Fatal("error generating random bytes")
	}

	dst := make([]byte, benchmarkBytes)
	if _, err := prng.Read(dst); err != nil {
		b.Fatal("error generating random bytes")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		And(dst, src)
	}
}

func BenchmarkDoubleXor(b *testing.B) {
	src := make([]byte, benchmarkBytes)
	if _, err := prng.Read(src); err != nil {
		b.Fatal("error generating random bytes")
	}

	src2 := make([]byte, benchmarkBytes)
	if _, err := prng.Read(src2); err != nil {
		b.Fatal("error generating random bytes")
	}

	dst := make([]byte, benchmarkBytes)
	if _, err := prng.Read(dst); err != nil {
		b.Fatal("error generating random bytes")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DoubleXor(dst, src, src2)
	}
}

func BenchmarkAndXor(b *testing.B) {
	src := make([]byte, benchmarkBytes)
	if _, err := prng.Read(src); err != nil {
		b.Fatal("error generating random bytes")
	}

	src2 := make([]byte, benchmarkBytes)
	if _, err := prng.Read(src2); err != nil {
		b.Fatal("error generating random bytes")
	}

	dst := make([]byte, benchmarkBytes)
	if _, err := prng.Read(dst); err != nil {
		b.Fatal("error generating random bytes")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AndXor(dst, src, src2)
	}
}

func BenchmarkConcurrentBitOp(b *testing.B) {
	src := make([]byte, benchmarkBytes)
	if _, err := prng.Read(src); err != nil {
		b.Fatal("error generating random bytes")
	}

	dst := make([]byte, benchmarkBytes)
	if _, err := prng.Read(dst); err != nil {
		b.Fatal("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentBitOp(Xor, dst, src)
	}
}

func BenchmarkConcurrentDoubleBitOp(b *testing.B) {
	src := make([]byte, benchmarkBytes)
	if _, err := prng.Read(src); err != nil {
		b.Fatal("error generating random bytes")
	}

	src2 := make([]byte, benchmarkBytes)
	if _, err := prng.Read(src2); err != nil {
		b.Fatal("error generating random bytes")
	}

	dst := make([]byte, benchmarkBytes)
	if _, err := prng.Read(dst); err != nil {
		b.Fatal("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentDoubleBitOp(AndXor, dst, src, src2)
	}
}
