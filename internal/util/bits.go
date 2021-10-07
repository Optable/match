package util

import (
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
)

var ErrByteLengthMissMatch = fmt.Errorf("provided bytes do not have the same length for bit operations")

// XorBytes XORS each byte from a with b and returns dst
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

// Inplace XorBytes XORS each byte from a with b and returns dst
// if a and b are the same length
func InPlaceXorBytes(a, dst []byte) error {
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := 0; i < n; i++ {
		dst[i] ^= a[i]
	}

	return nil
}

// AndBytes returns the binary AND of each byte in a and b
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

// InplaceAndBytes replaces the bytes in dst with the binary AND of
// each byte with the corresponding byte in a (if a and b are the
// same length).
func InPlaceAndBytes(a, dst []byte) error {
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := range dst {
		dst[i] = dst[i] & a[i]
	}

	return nil
}

// AndByte returns the binary AND of each byte in b with a.
func AndByte(a uint8, b []byte) []byte {
	if a == 1 {
		return b
	}

	return make([]byte, len(b))
}

// TestBitSetInByte returns 1 if bit i is set in a byte slice.
// it extracts bits from the least significant bit (i = 0) to the
// most significant bit (i = 7)
func TestBitSetInByte(b []byte, i int) byte {
	if b[i/8]&(1<<(i%8)) > 0 {
		return 1
	}
	return 0
}

// Transpose returns the transpose of a 2D slices of bytes
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

// Transpose3D returns the transpose of a 3D slices of bytes
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

// SampleRandomDenseBitMatrix fills each entry in the given 2D slices of bytes
// with pseudorandom bit values but leaves them densely encoded unlike
// SampleRandomBitMatrix.
func SampleRandomBitMatrix(prng io.Reader, row, col int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, row)
	for row := range matrix {
		matrix[row] = make([]uint8, (col+PadTill512(col))/8)
	}

	for row := range matrix {
		if _, err := prng.Read(matrix[row]); err != nil {
			return nil, err
		}
	}

	return matrix, nil
}

// PadTill512 returns the number of rows/columns to pad such that the number is a
// multiple of 512.
func PadTill512(m int) (pad int) {
	pad = 512 - (m % 512)
	if pad == 512 {
		pad = 0
	}
	return pad
}

// TransposeByteMatrix performs a concurrent cache-oblivious transpose on a byte matrix by first
// converting from bytes to uint64 (and padding as needed), performing the transpose on the uint64
// matrix and then converting back to bytes.
func TransposeByteMatrix(b [][]byte) (tr [][]byte) {
	return ByteMatrixFromUint64(ConcurrentTranspose(Uint64MatrixFromByte(b), runtime.NumCPU()))
}

// Uint64SliceFromByte converts a slice of bytes to a slice of uint64s.
// There must be a multiple of 8 bytes so they can be packed nicely into uint64.
func Uint64SliceFromByte(b []byte) (u []uint64) {
	u = make([]uint64, len(b)/8)
	for i := range u {
		u[i] = binary.LittleEndian.Uint64(b[i*8:])
	}

	return u
}

// ByteSliceFromUint64 extracts a slice of bytes from a slice of uint64.
func ByteSliceFromUint64(u []uint64) (b []byte) {
	b = make([]byte, len(u)*8)

	for i, e := range u {
		binary.LittleEndian.PutUint64(b[i*8:], e)
	}

	return b
}

// Uint64MatrixFromByte converts matrix of bytes to matrix of uint64s.
// pad is number of rows containing 0s which will be added to end of matrix.
// Assume each row contains 64 bytes (512 bits).
func Uint64MatrixFromByte(b [][]byte) (u [][]uint64) {
	pad := PadTill512(len(b))
	u = make([][]uint64, len(b)+pad)

	for i := 0; i < len(b); i++ {
		u[i] = Uint64SliceFromByte(b[i])
	}

	for j := 0; j < pad; j++ {
		u[len(b)+j] = make([]uint64, len(u[0]))
	}

	return u
}

// ByteMatrixFromUint64 converts matrix of uint64s to matrix of bytes.
// If any padding was added, it is left untouched.
func ByteMatrixFromUint64(u [][]uint64) (b [][]byte) {
	b = make([][]byte, len(u))

	for i, e := range u {
		b[i] = ByteSliceFromUint64(e)
	}

	return b
}
