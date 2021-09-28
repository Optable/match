package util

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

var prng = rand.New(rand.NewSource(time.Now().UnixNano()))

func sampleByteSlice(prng *rand.Rand, b []byte) (err error) {
	if _, err = prng.Read(b); err != nil {
		return nil
	}
	return nil
}

func sampleByteMatrix(prng *rand.Rand, b [][]byte, m, n int) (err error) {
	for r := range b {
		if _, err = prng.Read(b[r]); err != nil {
			return nil
		}
	}
	return nil
}

func sampleUint64Slice(prng *rand.Rand, u []uint64) {
	for i := range u {
		u[i] = prng.Uint64()
	}
}

func sampleUint64Matrix(prng *rand.Rand, u [][]uint64) {
	for i := range u {
		sampleUint64Slice(prng, u[i])
	}
}

// Note the double conversion of bytes to uint64s to bytes does
// result in added 0s.
func TestSliceConversions(t *testing.T) {
	lengths := []int{1, 7, 8, 9, 100, 1000}
	for _, l := range lengths {
		// Bytes to Uint64s
		b := make([]byte, l)
		sampleByteSlice(prng, b)
		u := Uint64SliceFromByte(b)
		bb := ByteSliceFromUint64(u)

		// test
		for i, e := range b {
			if e != bb[i] {
				t.Errorf("Byte-to-Uint64-to-Byte conversion did not result in identical slices")
			}
		}

		// Uint64s to Bytes
		u = make([]uint64, l)
		sampleUint64Slice(prng, u)
		b = ByteSliceFromUint64(u)
		uu := Uint64SliceFromByte(b)

		//test
		for i, e := range u {
			if e != uu[i] {
				t.Errorf("Uint64-to-Byte-to-Uint64 conversion did not result in identical slices")
			}
		}

	}
}

func TestXorUint64(t *testing.T) {
	lengths := []int{1, 7, 8, 9, 100, 1000}
	for _, l := range lengths {
		u := make([]uint64, l)
		sampleUint64Slice(prng, u)
		c := make([]uint64, l)
		copy(c, u) // make a copy to check later
		// create set to do XOR
		x := make([]uint64, l)
		for i := range x {
			x[i] = 0x5555555555555555 // 01010...01
		}
		XorUint64Slice(u, x)
		XorUint64Slice(u, x)
		// test
		for i, e := range c {
			if e != u[i] {
				t.Errorf("Doubly-XORed slice not identical to original")
			}
		}
	}
}

func TestAndUint64(t *testing.T) {
	lengths := []int{1, 7, 8, 9, 100, 1000}
	for _, l := range lengths {
		u := make([]uint64, l)
		sampleUint64Slice(prng, u)
		c := make([]uint64, l)
		copy(c, u) // make a copy to check later
		// create set to do AND
		x := make([]uint64, l)
		for i := range x {
			x[i] = 0x5555555555555555 // 01010...01
		}
		AndUint64Slice(u, x)
		AndUint64Slice(x, c)
		// test
		for i, e := range u {
			if e != x[i] {
				t.Errorf("Duplicate AND slices not identical")
			}
		}
	}
}

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

func TestOldTranspose(t *testing.T) {

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
