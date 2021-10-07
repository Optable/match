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
		u := Uint64SliceFromByte(b)
		bb := ByteSliceFromUint64(u)

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
		b := ByteSliceFromUint64(u)
		uu := Uint64SliceFromByte(b)

		//test
		for i, e := range u {
			if e != uu[i] {
				t.Errorf("Uint64-to-Byte-to-Uint64 conversion did not result in identical slices")
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
		prng.Read(b[m][0])
		prng.Read(b[m][1])
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
		prng.Read(b[m])
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
	prng.Read(a)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		XorBytes(a, a)
	}
}

func BenchmarkInplaceXorBytes(b *testing.B) {
	a := make([]byte, 10000)
	prng.Read(a)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InPlaceXorBytes(a, a)
	}
}
