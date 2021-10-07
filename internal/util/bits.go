package util

import (
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
	"sync"
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
func InPlaceXorBytes(dst, a []byte) error {
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := 0; i < n; i++ {
		dst[i] ^= a[i]
	}

	return nil
}

func ConcurrentInPlaceXorBytes(dst, a []byte) error {
	const blockSize int = 16384 // half of what L2 cache can hold
	nworkers := runtime.NumCPU()
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	if n < blockSize {
		return InPlaceXorBytes(dst, a)
	}

	// determine number of blocks to split original matrix
	nblks := n / blockSize
	if n%blockSize != 0 {
		nblks += 1
	}
	ch := make(chan int, nblks)
	for i := 0; i < nblks; i++ {
		ch <- i
	}
	close(ch)

	// Run a worker pool
	var wg sync.WaitGroup
	wg.Add(nworkers)
	for w := 0; w < nworkers; w++ {
		go func() {
			defer wg.Done()
			for blk := range ch {
				step := blockSize * blk
				if blk == nblks-1 { // last block
					for i := step; i < n; i++ {
						dst[i] ^= a[i]
					}
				} else {
					for i := step; i < step+blockSize; i++ {
						dst[i] ^= a[i]
					}
				}
			}
		}()
	}

	wg.Wait()

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
func SampleRandomDenseBitMatrix(prng io.Reader, row, col int) ([][]uint8, error) {
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

// SampleRandomBitMatrix fills each entry in the given 2D slices of bytes
// with pseudorandom bit values
func SampleRandomBitMatrix(prng io.Reader, row, col int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, row)
	for row := range matrix {
		matrix[row] = make([]uint8, col)
	}

	for row := range matrix {
		if err := SampleBitSlice(prng, matrix[row]); err != nil {
			return nil, err
		}
	}

	return matrix, nil
}

// SampleBitSlice returns a slice of bytes of pseudorandom bits
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

// ExtractBytesToBits returns a byte array of bits from src
// if len(dst) < len(src) * 8, nothing will be done
func ExtractBytesToBits(src, dst []byte) {
	if len(dst) != len(src)*8 {
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
