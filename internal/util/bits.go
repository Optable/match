package util

import (
	"encoding/binary"
	"fmt"
	"math/rand"
)

var ErrByteLengthMissMatch = fmt.Errorf("provided bytes do not have the same length for XOR operations")

// XorBytes xors each byte from a with b and returns dst
// if a and b are the same length
func XorBytes(a, b []byte) (dst []byte, err error) {
	var n = len(b)
	if n != len(a) {
		return nil, ErrByteLengthMissMatch
	}

	dst = make([]byte, n)

	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}

	return
}

// Inplace XorBytes xors each byte from a with b and returns dst
// if a and b are the same length
func InPlaceXorBytes(a, dst []byte) error {
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ dst[i]
	}

	return nil
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

// InplaceAndBytes returns the binary and of each byte in a and b
// if a and b are the same length
func InPlaceAndBytes(a, dst []byte) error {
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := range dst {
		dst[i] = a[i] & dst[i]
	}

	return nil
}

func AndByte(a uint8, b []byte) (dst []byte) {
	dst = make([]byte, len(b))

	for i := range b {
		dst[i] = a & b[i]
	}

	return
}

func InPlaceAndByte(a uint8, dst []byte) {
	for i := range dst {
		dst[i] = a & dst[i]
	}
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
	// read up to len(b) + 1 pseudorandom bits
	t := make([]byte, len(b)/8+1)
	if _, err = prng.Read(t); err != nil {
		return nil
	}

	// extract all bits into b
	ExtractBytesToBits(t, b)

	return nil
}

// sampleRandomTall fills an m by 8 uint64 matrix (512 bits wide) with
// pseudorandom uint64.
func SampleRandomTall(r *rand.Rand, m int) [][]uint64 {
	// instantiate matrix
	matrix := make([][]uint64, m)

	for row := range matrix {
		matrix[row] = make([]uint64, 8)
		for c := 0; c < 8; c++ {
			matrix[row][c] = r.Uint64()
		}
	}

	return matrix
}

// SampleRandomWide fills a 512 by n uint64 matrix (512 bits tall) with
// pseudorandom uint64.
func SampleRandomWide(r *rand.Rand, n int) [][]uint64 {
	// instantiate matrix
	matrix := make([][]uint64, 512)

	for row := range matrix {
		matrix[row] = make([]uint64, n)
		for c := 0; c < n; c++ {
			matrix[row][c] = r.Uint64()
		}
	}

	return matrix
}

// ExtractBytesToBits returns a byte array of bits from src
// if len(dst) < len(src) * 8, nothing will be done
func ExtractBytesToBits(src, dst []byte) {
	if len(dst) > len(src)*8 {
		return
	}

	var i int
	for _, _byte := range src[:len(src)-1] {
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

	// handle the last byte
	for i = 0; i < len(dst)%8; i++ {
		dst[(len(src)-1)*8+i] = uint8((src[len(src)-1] >> i) & 0x01)
	}
}

// Uint64Slice converts a slice of bytes to a slice of uint64s.
// Note: additional 0's will be appended to the byte slice to
// ensure it has a multiple of 8 elements.
func Uint64Slice(b []byte) (u []uint64) {
	// expand byte slice to a multiple of 8
	var x int
	if len(b)%8 != 0 {
		x = 8 - (len(b) % 8)
	}

	b = append(b, make([]byte, x)...)

	u = make([]uint64, len(b)/8)
	for i := 0; i < len(b); i += 8 {
		u[i/8] = binary.LittleEndian.Uint64(b[i:])
	}

	return u
}

// ByteSlice extracts a slice of bytes from a slice of uint64.
func ByteSlice(u []uint64) (b []byte) {
	b = make([]byte, len(u)*8)

	for i, e := range u {
		binary.LittleEndian.PutUint64(b[i*8:], e)
	}

	return b
}

// Uint64Matrix converts matrix of bytes to matrix of uint64s.
func Uint64Matrix(b [][]byte) (u [][]uint64) {
	u = make([][]uint64, len(b))

	for i, e := range b {
		u[i] = Uint64Slice(e)
	}

	return u
}

// ByteMatrix converts matrix of uint64s to matrix of bytes.
func ByteMatrix(u [][]uint64) (b [][]byte) {
	b = make([][]byte, len(u))

	for i, e := range u {
		b[i] = ByteSlice(e)
	}

	return b
}

// XorUint64 performs the binary XOR of each uint64 in u and w
// in-place as long as the slices are of the same length. u is
// the modified slice.
func XorUint64(u, w []uint64) error {
	if len(u) != len(w) {
		return fmt.Errorf("provided slices do not have the same length for XOR operations")
	}

	for i := range u {
		u[i] ^= w[i]
	}

	return nil
}

// AndUint64 performs the binary AND of each uint64 in u and w
// in-place as long as the slices are of the same length. u is
// the modified slice.
func AndUint64(u, w []uint64) error {
	if len(u) != len(w) {
		return fmt.Errorf("provided slices do not have the same length for AND operations")
	}

	for i := range u {
		u[i] &= w[i]
	}

	return nil
}
