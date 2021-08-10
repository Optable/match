package util

import (
	"fmt"
	"math/rand"

	"github.com/bits-and-blooms/bitset"
)

var ErrByteLengthMissMatch = fmt.Errorf("provided bytes do not have the same length for XOR operations")

// XorBytes xors each byte from a with b and returns dst
// if a and b are the same length
func XorBytes(a, b []byte) (dst []byte, err error) {
	n := len(b)
	if n != len(a) {
		return nil, ErrByteLengthMissMatch
	}

	dst = make([]byte, n)

	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}

	return
}

// AndBytes returns the binary and of each byte in a and b
// if a and b are the same length
func AndBytes(a, b []byte) (dst []byte, err error) {
	n := len(b)
	if n != len(a) {
		return nil, ErrByteLengthMissMatch
	}

	dst = make([]byte, n)

	for i := 0; i < n; i++ {
		dst[i] = a[i] & b[i]
	}

	return
}

// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
func Transpose(matrix [][]uint8) [][]uint8 {
	n := len(matrix)
	tr := make([][]uint8, len(matrix[0]))

	for row := range tr {
		tr[row] = make([]uint8, n)
		for col := range tr[row] {
			tr[row][col] = matrix[col][row]
		}
	}
	return tr
}

// Transpose3D returns the transpose of a 3D slices of uint8
// from (m x 2 x k) to (k x 2 x m)
func Transpose3D(matrix [][][]uint8) [][][]uint8 {
	n := len(matrix)
	tr := make([][][]uint8, len(matrix[0][0]))

	for row := range tr {
		tr[row] = make([][]uint8, len(matrix[0]))
		for b := range tr[row] {
			tr[row][b] = make([]uint8, n)
			for col := range tr[row][b] {
				tr[row][b][col] = matrix[col][b][row]
			}
		}
	}
	return tr
}

// SampleRandomBitMatrix fills each entry in the given 2D slice with
// pseudorandom bit values in a bitset
func SampleRandomBitMatrix(r *rand.Rand, m, n int) []*bitset.BitSet {
	// instantiate matrix
	matrix := make([]*bitset.BitSet, m)

	for row := range matrix {
		matrix[row] = SampleBitSlice(r, n)
	}

	return matrix
}

// SampleBitSlice returns a bitset of pseudorandom bits
func SampleBitSlice(r *rand.Rand, n int) *bitset.BitSet {
	var numInts int
	if n%64 != 0 {
		numInts = n/64 + 1
	} else {
		numInts = n / 64
	}
	seedInts := make([]uint64, numInts)

	for i := range seedInts {
		seedInts[i] = r.Uint64()
	}

	return bitset.From(seedInts)
}

// ExtractBytesToBits returns a byte array of bits from src
// if len(dst) < len(src) * 8, nothing will be done
func ExtractBytesToBits(src, dst []byte) {
	if len(dst) < len(src)*8 {
		return
	}

	var i int
	for _, _byte := range src {
		dst[i] = uint8(_byte & 0x01)
		dst[i+1] = uint8((_byte >> 1) & 0x01)
		dst[i+2] = uint8((_byte >> 2) & 0x01)
		dst[i+3] = uint8((_byte >> 3) & 0x01)
		dst[i+4] = uint8((_byte >> 4) & 0x01)
		dst[i+5] = uint8((_byte >> 5) & 0x01)
		dst[i+6] = uint8((_byte >> 6) & 0x01)
		dst[i+7] = uint8((_byte >> 7) & 0x01)
		i += 8
	}
}
