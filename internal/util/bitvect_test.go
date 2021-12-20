package util

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

const benchmarkMatrixLength = 1 << 20

// sampleRandomTall fills an m by 64 byte matrix (512 bits wide) with
// pseudorandom bytes.
func sampleRandomTall(m int) [][]byte {
	// instantiate matrix
	matrix := make([][]byte, m)

	for row := range matrix {
		matrix[row] = make([]byte, 64)
		rand.Read(matrix[row])
	}

	return matrix
}

// sampleRandomWide fills a 512 by n byte matrix (512 bits tall) with
// pseudorandom bytes.
func sampleRandomWide(n int) [][]byte {
	// instantiate matrix
	matrix := make([][]byte, 512)

	for row := range matrix {
		matrix[row] = make([]byte, n)
		rand.Read(matrix[row])
	}

	return matrix
}

// sampleZebraBlock creates a 512x512 bit block where every bit position
// alternates between 0 and 1. When transposed, this block should
// consists of rows of all 0s alternating with rows of all 1s.
func sampleZebraBlock() BitVect {
	zebraBlock2D := make([][]byte, 512)
	var b BitVect
	for row := range zebraBlock2D {
		zebraBlock2D[row] = make([]byte, 64)
		for c := 0; c < 64; c++ {
			zebraBlock2D[row][c] = 0b01010101
		}
	}
	b.unravelTall(zebraBlock2D, 0)
	return b
}

// Property-based tests

type tallMatrix struct {
	matrix [][]byte
}

// Generate creates a struct containing two duplicate
// copies of a pseudorandom tall matrix which has 512
// columns and a multiple of 512 rows.
func (tallMatrix) Generate(r *rand.Rand, size int) reflect.Value {
	var tall tallMatrix
	tall.matrix = sampleRandomTall(Pad(size, 512))
	return reflect.ValueOf(tall)
}

func TestUnReRavelTall(t *testing.T) {
	unravel := func(t tallMatrix) bool {
		var b BitVect
		// determine number of blocks to split original matrix (m x 64)
		nblks := len(t.matrix) / 512
		for i := 0; i < nblks; i++ {
			b.unravelTall(t.matrix, i)
		}

		for r := range t.matrix {
			for i := 0; i < 512; i++ {
				if IsBitSet(t.matrix[r], i) != IsBitSetUint64(b.set[:], (r*512)+i) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(unravel, nil); err != nil {
		t.Errorf("unraveling of a tall matrix was incorrect: %v", err)
	}

	ravel := func(t tallMatrix) bool {
		var b BitVect
		// determine number of blocks to split original matrix (m x 64)
		nblks := len(t.matrix) / 512
		// create empty matrix to ravel into
		scratch := make([][]byte, len(t.matrix))
		for r := range scratch {
			scratch[r] = make([]byte, 512)
		}

		for i := 0; i < nblks; i++ {
			b.unravelTall(t.matrix, i)
			b.ravelToTall(scratch, i)
		}

		for r := range t.matrix {
			for i := 0; i < 512; i++ {
				if IsBitSet(t.matrix[r], i) != IsBitSet(scratch[r], i) {
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
	correct := func(t tallMatrix) bool {
		tr := ConcurrentTransposeTall(t.matrix)
		for r := range t.matrix {
			for i := 0; i < 512; i++ {
				if IsBitSet(t.matrix[r], i) != IsBitSet(tr[i], r) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(correct, nil); err != nil {
		t.Errorf("transpose of a tall matrix was incorrect: %v", err)
	}

	involutory := func(t tallMatrix) bool {
		tr := ConcurrentTransposeTall(t.matrix)
		dtr := ConcurrentTransposeTall(tr)
		for r := range t.matrix {
			for i := 0; i < 512; i++ {
				if IsBitSet(t.matrix[r], i) != IsBitSet(dtr[r], i) {
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

// Generate creates a struct containing two duplicate
// copies of a pseudorandom wide matrix which has 512
// rows and a multiple of 512 columns.
func (wideMatrix) Generate(r *rand.Rand, size int) reflect.Value {
	var wide wideMatrix
	wide.matrix = sampleRandomWide(Pad(size, 512))
	return reflect.ValueOf(wide)
}

func TestUnReRavelWide(t *testing.T) {
	unravel := func(t wideMatrix) bool {
		var b BitVect
		// determine number of blocks to split original matrix (512 x n)
		nblks := len(t.matrix[0]) / 512
		for i := 0; i < nblks; i++ {
			b.unravelWide(t.matrix, i)
		}

		for r := 0; r < 512; r++ {
			for i := range t.matrix[r] {
				if IsBitSet(t.matrix[r], i) != IsBitSetUint64(b.set[:], (r*512)+i) {
					return false
				}
			}
		}
		return true
	}

	if err := quick.Check(unravel, nil); err != nil {
		t.Errorf("unraveling of a wide matrix was incorrect: %v", err)
	}

	ravel := func(t wideMatrix) bool {
		var b BitVect
		// determine number of blocks to split original matrix (512 x n)
		nblks := len(t.matrix[0]) / 512
		// create empty matrix to ravel into
		scratch := make([][]byte, 512)
		for r := range scratch {
			scratch[r] = make([]byte, len(t.matrix[r]))
		}

		for i := 0; i < nblks; i++ {
			b.unravelWide(t.matrix, i)
			b.ravelToWide(scratch, i)
		}

		for r := 0; r < 512; r++ {
			for i := range t.matrix[r] {
				if IsBitSet(t.matrix[r], i) != IsBitSet(scratch[r], i) {
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
	correct := func(t wideMatrix) bool {
		tr := ConcurrentTransposeWide(t.matrix)
		for r, row := range t.matrix {
			for i := 0; i < 512; i++ {
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

	involutory := func(t wideMatrix) bool {
		tr := ConcurrentTransposeWide(t.matrix)
		dtr := ConcurrentTransposeWide(tr)
		for r := 0; r < 512; r++ {
			for i := range t.matrix[r] {
				if IsBitSet(t.matrix[r], i) != IsBitSet(dtr[r], i) {
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
	for i := 0; i < 512; i++ {
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
	matrix := sampleRandomTall(benchmarkMatrixLength)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentTransposeTall(matrix)
	}
}

func BenchmarkConcurrentTransposeWide(b *testing.B) {
	matrix := sampleRandomWide(benchmarkMatrixLength)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentTransposeWide(matrix)
	}
}
