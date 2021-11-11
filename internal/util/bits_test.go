package util

import (
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
			if !BitSetInByte(b, i) {
				t.Fatalf("bit extraction failed")
			}
		} else {
			if BitSetInByte(b, i) {
				t.Fatalf("bit extraction failed")
			}
		}
	}

	b = []byte{161}
	for i := 0; i < 8; i++ {
		if i == 0 || i == 7 || i == 5 {
			if !BitSetInByte(b, i) {
				t.Fatalf("bit extraction failed")
			}
		} else {
			if BitSetInByte(b, i) {
				t.Fatalf("bit extraction failed")
			}
		}

	}
}

// Note the double conversion of bytes to uint64s to bytes does
// result in added 0s.
// Only tested on AMD64.
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

func BenchmarkUnsafeInPlaceXorBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Xor(a, a)
	}
}

func BenchmarkConcurrentUnsafeBitOperation(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentBitOp(Xor, a, a)
	}
}
