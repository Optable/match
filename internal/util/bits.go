package util

import (
	"encoding/binary"
	"fmt"
	"io"
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
		dst[i] ^= a[i]
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
		dst[i] = dst[i] & a[i]
	}

	return nil
}

// TestBitSetInByte returns true if bit i is set in a byte slice.
// it extracts bits from the least significant bit (i = 0) to the
// most significant bit (i = 7)
func TestBitSetInByte(b []byte, i int) byte {
	if b[i/8]&(128>>(i%8)) > 0 {
		return 1
	}
	return 0
}

func AndByte(a uint8, b []byte) []byte {
	if a == 1 {
		return b
	}

	return make([]byte, len(b))
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
func SampleRandomBitMatrix(prng io.Reader, row, col int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, row)
	for row := range matrix {
		matrix[row] = make([]uint8, (col+RowsToPad(col))/8)
	}

	for row := range matrix {
		if _, err := prng.Read(matrix[row]); err != nil {
			return nil, err
		}
	}

	return matrix, nil
}

func SampleZerosBitMatrix(prng io.Reader, row, col int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, row)
	for row := range matrix {
		matrix[row] = make([]uint8, (col+RowsToPad(col))/8)
	}

	return matrix, nil
}

func SampleOnesBitMatrix(prng io.Reader, row, col int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, row)
	for row := range matrix {
		matrix[row] = make([]uint8, (col+RowsToPad(col))/8)
	}

	for row := range matrix {
		for col := range matrix[row] {
			matrix[row][col] = byte(1)
		}
	}

	return matrix, nil
}

// SampleBitSlice returns a slice of uint8 of pseudorandom bits
// prng is a reader from either crypto/rand.Reader
// or math/rand.Rand
func SampleBitSlice(prng io.Reader, b []uint8) (err error) {
	// read up to len(b) + 1 pseudorandom bits
	t := make([]byte, len(b)/8)
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
	if len(dst) != len(src)*8 {
		fmt.Println(len(dst), len(src))
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

// RowsToPad returns the number of rows to pad such that the number of rows is
// a multiple of 512.
func RowsToPad(m int) (pad int) {
	pad = 512 - (m % 512)
	if pad == 512 {
		pad = 0
	}
	return pad
}

// TransposeByteMatrix performs a concurrent cache-oblivious transpose on a byte matrix by first
// converting from bytes to uint64 (and padding as needed), performing the tranpose on the uint64
// matrix and then converting back to bytes.
func TransposeByteMatrix(b [][]byte) (tr [][]byte) {
	return ByteMatrixFromUint64(ConcurrentTranspose(Uint64MatrixFromByte(b), 6))
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
		binary.BigEndian.PutUint64(b[i*8:], e)
	}

	return b
}

// Uint64MatrixFromByte converts matrix of bytes to matrix of uint64s.
// pad is number of rows containing 0s which will be added to end of matrix.
// Assume each row contains 64 bytes (512 bits).
func Uint64MatrixFromByte(b [][]byte) (u [][]uint64) {
	pad := RowsToPad(len(b))
	u = make([][]uint64, len(b)+pad)

	for i := 0; i < len(b); i++ {
		u[i] = Uint64SliceFromByte(b[i])
	}

	for j := 0; j < pad; j++ {
		u[len(b)+j] = make([]uint64, len(u[0]))
	}

	return u
}

// FromUint64ToByteMatrix converts matrix of uint64s to matrix of bytes.
// If any padding was added, it is left untouched.
func ByteMatrixFromUint64(u [][]uint64) (b [][]byte) {
	b = make([][]byte, len(u))

	for i, e := range u {
		b[i] = ByteSliceFromUint64(e)
	}

	return b
}

// XorUint64Slice performs the binary XOR of each uint64 in u and w
// in-place as long as the slices are of the same length. u is
// the modified slice.
func XorUint64Slice(u, w []uint64) error {
	if len(u) != len(w) {
		return fmt.Errorf("provided slices do not have the same length for XOR operations")
	}

	for i := range u {
		u[i] ^= w[i]
	}

	return nil
}

// AndUint64Slice performs the binary AND of each uint64 in u and w
// in-place as long as the slices are of the same length. u is
// the modified slice.
func AndUint64Slice(u, w []uint64) error {
	if len(u) != len(w) {
		return fmt.Errorf("provided slices do not have the same length for AND operations")
	}

	for i := range u {
		u[i] &= w[i]
	}

	return nil
}
