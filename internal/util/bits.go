package util

import (
	"encoding/binary"
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

// XORs two BitSets if they are the same length
func XorBitsets(a, b *bitset.BitSet) (*bitset.BitSet, error) {
	n := b.Len()
	if n != a.Len() {
		return nil, ErrByteLengthMissMatch
	}

	return a.SymmetricDifference(b), nil
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

// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
func ContiguousTranspose(matrix [][]uint8) [][]uint8 {
	m := len(matrix)
	k := len(matrix[0])
	tr := make([][]uint8, k)
	longRow := make([]uint8, m*k)

	for x := range longRow {
		longRow[x] = matrix[x%m][x/m]
	}

	for i := range tr {
		tr[i] = longRow[i*m : (i+1)*m]
	}

	return tr
}

// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
// This is MORE efficient that the other version
func ContiguousTranspose2(matrix [][]uint8) [][]uint8 {
	m := len(matrix)
	k := len(matrix[0])
	tr := make([][]uint8, k)
	longRow := make([]uint8, m*k)

	for i := 0; i < m; i++ {
		for j := 0; j < k; j++ {
			longRow[j*m+i] = matrix[i][j]
		}
	}

	for i := range tr {
		tr[i] = longRow[i*m : (i+1)*m]
	}

	return tr
}

/*
// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
func TransposeInPlace(matrix [][]uint8) [][]uint8 {
	m := len(matrix)
	n := len(matrix[0])

	// determine by how much to expand matrix to make it square
	longSide := m
	if n > m {
		longSide = n
	}

	if longSide < 2 {
		// tiny matrix
		longSide = 2
	} else {
		// otherwise we want divible by 4
		longSide += 4 - (longSide % 4)
	}

	// make expanded square matrix
	sqMatrix := make([][]uint8, longSide)
	for row := range sqMatrix {
		sqMatrix[row] = make([]uint8, longSide)
		if row < m {
			copy(sqMatrix[row], matrix[row])
		}
	}

	// set initial value for recursion
	for boxSize := longSide / 2; boxSize > 0; boxSize /= 2 {

	}
	return tr
}
*/
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

// SampleRandomBitSetMatrix fills each entry in the given 2D slice with
// pseudorandom bit values in a bitset
func SampleRandomBitSetMatrix(r *rand.Rand, m, n int) []*bitset.BitSet {
	// instantiate matrix
	matrix := make([]*bitset.BitSet, m)

	for row := range matrix {
		matrix[row] = SampleBitSetSlice(r, n)
	}

	return matrix
}

// SampleBitSetSlice returns a bitset of pseudorandom bits
func SampleBitSetSlice(r *rand.Rand, n int) *bitset.BitSet {
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

// SampleRandomBitMatrix fills each entry in the given 2D slices of uint8
// with pseudorandom bit values
func SampleRandomBitMatrix(r *rand.Rand, m, k int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, m)
	for row := range matrix {
		matrix[row] = make([]uint8, k)
	}

	for row := range matrix {
		if err := SampleBitSlice(r, matrix[row]); err != nil {
			return nil, err
		}
	}

	return matrix, nil
}

// SampleBitSlice returns a slice of uint8 of pseudorandom bits
func SampleBitSlice(prng *rand.Rand, b []uint8) (err error) {
	// read up to len(b) pseudorandom bits
	t := make([]byte, len(b)/8)
	if _, err = prng.Read(t); err != nil {
		return nil
	}

	// extract all bits into b
	ExtractBytesToBits(t, b)

	return nil
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

// Extract a slice of bytes from a BitSet
func BitSetToBytes(bset *bitset.BitSet) []byte {
	b := make([]byte, len(bset.Bytes())*8)

	for i, x := range bset.Bytes() {
		binary.LittleEndian.PutUint64(b[i*8:], x)
	}

	return b
}

// Extract a matrix of bytes from slices of BitSets
func BitSetsToByteMatrix(bsets []*bitset.BitSet) [][]byte {
	b := make([][]byte, len(bsets))

	for i, x := range bsets {
		b[i] = BitSetToBytes(x)
	}

	return b
}

// Convert slice of bytes to BitSet
// Note: additional 0's will be appended to the byte slice
//       to ensure it has a multiple of 8 elements
func BytesToBitSet(b []byte) *bitset.BitSet {
	// expand byte slice to a multiple of 8
	var x int
	if len(b)%8 != 0 {
		x = 8 - (len(b) % 8)
	}

	b = append(b, make([]byte, x)...)

	b64 := make([]uint64, len(b)/8)
	for i := 0; i < len(b); i += 8 {
		b64[i/8] = binary.LittleEndian.Uint64(b[i:])
	}

	return bitset.From(b64)
}

// Convert matrix of bytes to slices of BitSets
func ByteMatrixToBitsets(b [][]byte) []*bitset.BitSet {
	bsets := make([]*bitset.BitSet, len(b))

	for i, x := range b {
		bsets[i] = BytesToBitSet(x)
	}

	return bsets
}

// m x k to k x m
func TransposeBitSets(bmat []*bitset.BitSet) []*bitset.BitSet {
	m := uint(len(bmat))
	k := bmat[0].Len()

	// setup new matrix of BitSets to hold transposed values
	transposed := make([]*bitset.BitSet, k)
	for row := range transposed {
		transposed[row] = bitset.New(m)
	}

	// iterate through original BitSets
	for i, b := range bmat {
		for j := 0; uint(j) < k; j++ {
			if b.Test(uint(j)) {
				transposed[j].Set(uint(i))
			}
		}
	}
	return transposed
}
