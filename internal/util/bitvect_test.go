package util

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

const benchmarkMatrixLength = 1 << 20

// isBitSetUint64 returns true if bit i is set in a uint64 slice.
// It extracts bits from the least significant bit (i = 0) to the
// most significant bit (i = 63).
func isBitSetUint64(u []uint64, i int) bool {
	return u[i/64]&(1<<(i%64)) > 0
}

// sampleRandomTall fills an m by 64 byte matrix (512 bits wide) with
// pseudorandom bytes.
func sampleRandomTall(m int, r *rand.Rand) [][]byte {
	// instantiate matrix
	matrix := make([][]byte, m)

	for row := range matrix {
		matrix[row] = make([]byte, bitVectWidth/8)
		r.Read(matrix[row])
	}

	return matrix
}

// sampleRandomWide fills a 512 by n byte matrix (512 bits tall) with
// pseudorandom bytes.
func sampleRandomWide(n int, r *rand.Rand) [][]byte {
	// instantiate matrix
	matrix := make([][]byte, bitVectWidth)

	for row := range matrix {
		matrix[row] = make([]byte, n)
		r.Read(matrix[row])
	}

	return matrix
}

// sampleZebraBlock creates a 512x512 bit block where every bit position
// alternates between 0 and 1. When transposed, this block should
// consists of rows of all 0s alternating with rows of all 1s.
func sampleZebraBlock() BitVect {
	zebraBlock2D := make([][]byte, bitVectWidth)
	var b BitVect
	for row := range zebraBlock2D {
		zebraBlock2D[row] = make([]byte, bitVectWidth/8)
		for c := 0; c < (bitVectWidth / 8); c++ {
			zebraBlock2D[row][c] = 0b01010101
		}
	}
	b.unravelTall(zebraBlock2D, 0)
	return b
}

type tallMatrix struct {
	matrix [][]byte
}

// Generate creates a struct containing a pseudorandom
// tall matrix which has 512 columns and a multiple of
// 512 rows.
func (tallMatrix) Generate(r *rand.Rand, unusedSizeHint int) reflect.Value {
	var tall tallMatrix
	size := 1 + r.Intn(5*bitVectWidth) // a max of 6 blocks
	tall.matrix = sampleRandomTall(Pad(size, bitVectWidth), r)
	return reflect.ValueOf(tall)
}

func TestUnReRavelTall(t *testing.T) {
	unravel := func(m tallMatrix) bool {
		var b BitVect
		// determine number of blocks to split original matrix (m x 64)
		nblks := len(m.matrix) / bitVectWidth
		for i := 0; i < nblks; i++ {
			// unravel block
			b.unravelTall(m.matrix, i)
			// and check
			for r := 0; r < bitVectWidth; r++ {
				for e := 0; e < bitVectWidth; e++ {
					if IsBitSet(m.matrix[(i*bitVectWidth)+r], e) != isBitSetUint64(b.set[:], (r*bitVectWidth)+e) {
						return false
					}
				}
			}
		}

		return true
	}

	if err := quick.Check(unravel, nil); err != nil {
		t.Errorf("unraveling of a tall matrix was incorrect: %v", err)
	}
	ravel := func(m tallMatrix) bool {
		var b BitVect
		// determine number of blocks to split original matrix (m x 64)
		nblks := len(m.matrix) / bitVectWidth
		// create empty matrix to ravel into
		scratch := make([][]byte, len(m.matrix))
		for r := range scratch {
			scratch[r] = make([]byte, bitVectWidth/8)
		}

		for i := 0; i < nblks; i++ {
			b.unravelTall(m.matrix, i)
			b.ravelToTall(scratch, i)
		}

		for r := range m.matrix {
			for i := 0; i < bitVectWidth; i++ {
				if IsBitSet(m.matrix[r], i) != IsBitSet(scratch[r], i) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(ravel, nil); err != nil {
		t.Errorf("raveling of a tall matrix was incorrect: %v", err)
	}
}

func TestTransposeTall(t *testing.T) {

	correct := func(m tallMatrix) bool {
		tr := ConcurrentTransposeTall(m.matrix)
		for r := range m.matrix {
			for i := 0; i < bitVectWidth; i++ {
				if IsBitSet(m.matrix[r], i) != IsBitSet(tr[i], r) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(correct, nil); err != nil {
		t.Errorf("transpose of a tall matrix was incorrect: %v", err)
	}

	involutory := func(m tallMatrix) bool {
		tr := ConcurrentTransposeTall(m.matrix)
		dtr := ConcurrentTransposeWide(tr)
		for r := range m.matrix {
			for i := 0; i < bitVectWidth; i++ {
				if IsBitSet(m.matrix[r], i) != IsBitSet(dtr[r], i) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(involutory, nil); err != nil {
		t.Errorf("double transpose of a tall matrix did not result in original matrix: %v", err)
	}
}

type wideMatrix struct {
	matrix [][]byte
}

// Generate creates a struct containing a pseudorandom
// wide matrix which has 512 rows and a multiple of
// 512 columns.
func (wideMatrix) Generate(r *rand.Rand, unusedSizeHint int) reflect.Value {
	var wide wideMatrix
	size := 1 + r.Intn(5*bitVectWidth) // a max of 6 blocks
	wide.matrix = sampleRandomWide(Pad(size, bitVectWidth), r)
	return reflect.ValueOf(wide)
}

func TestUnReRavelWide(t *testing.T) {
	unravel := func(m wideMatrix) bool {
		var b BitVect
		// determine number of blocks to split original matrix (512 x n)
		nblks := len(m.matrix[0]) / (bitVectWidth / 8)
		for i := 0; i < nblks; i++ {
			// unravel block
			b.unravelWide(m.matrix, i)
			// and check
			for r := 0; r < bitVectWidth; r++ {
				for e := 0; e < bitVectWidth; e++ {
					if IsBitSet(m.matrix[r], (i*bitVectWidth)+e) != isBitSetUint64(b.set[:], (r*bitVectWidth)+e) {
						return false
					}
				}
			}
		}

		return true
	}

	if err := quick.Check(unravel, nil); err != nil {
		t.Errorf("unraveling of a wide matrix was incorrect: %v", err)
	}

	ravel := func(m wideMatrix) bool {
		var b BitVect
		// determine number of blocks to split original matrix (512 x n)
		nblks := len(m.matrix[0]) / (bitVectWidth / 8)
		// create empty matrix to ravel into
		scratch := make([][]byte, bitVectWidth)
		for r := range scratch {
			scratch[r] = make([]byte, len(m.matrix[r]))
		}

		for i := 0; i < nblks; i++ {
			b.unravelWide(m.matrix, i)
			b.ravelToWide(scratch, i)
		}

		for r := 0; r < bitVectWidth; r++ {
			for i := range m.matrix[r] {
				if IsBitSet(m.matrix[r], i) != IsBitSet(scratch[r], i) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(ravel, nil); err != nil {
		t.Errorf("raveling of a wide matrix was incorrect: %v", err)
	}
}

func TestTransposeWide(t *testing.T) {
	correct := func(m wideMatrix) bool {
		tr := ConcurrentTransposeWide(m.matrix)
		for r, row := range m.matrix {
			for i := 0; i < bitVectWidth; i++ {
				if IsBitSet(row, i) != IsBitSet(tr[i], r) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(correct, nil); err != nil {
		t.Errorf("transpose of a wide matrix was incorrect: %v", err)
	}

	involutory := func(m wideMatrix) bool {
		tr := ConcurrentTransposeWide(m.matrix)
		dtr := ConcurrentTransposeTall(tr)
		for r := 0; r < bitVectWidth; r++ {
			for i := range m.matrix[r] {
				if IsBitSet(m.matrix[r], i) != IsBitSet(dtr[r], i) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(involutory, nil); err != nil {
		t.Errorf("double transpose of a wide matrix did not result in original matrix: %v", err)
	}
}

func TestIfLittleEndianTranspose(t *testing.T) {
	tr := sampleZebraBlock()
	// 0101....
	// 0101....
	// 0101....
	tr.transpose()
	// If Little Endian, we expect the resulting rows to be
	// 1111....
	// 0000....
	// 1111....

	// check if Little Endian
	for i := 0; i < bitVectWidth; i++ {
		if i%2 == 1 { // odd
			if tr.set[i*8] != 0 {
				t.Fatalf("error: transpose is Big Endian")
			}
		} else {
			if tr.set[i*8] != 0xFFFFFFFFFFFFFFFF {
				t.Fatalf("error: transpose is Big Endian")
			}
		}
	}
}

func BenchmarkConcurrentTransposeTall(b *testing.B) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	matrix := sampleRandomTall(benchmarkMatrixLength, r)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentTransposeTall(matrix)
	}
}

func BenchmarkConcurrentTransposeWide(b *testing.B) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	matrix := sampleRandomWide(benchmarkMatrixLength, r)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentTransposeWide(matrix)
	}
}
