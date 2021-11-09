package util

import (
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/alecthomas/unsafeslice"
)

var ErrByteLengthMissMatch = fmt.Errorf("provided bytes do not have the same length for bit operations")

// Xor casts the first part of the byte slice (length divisible
// by 8) into uint64 and then performs XOR on the slice of uint64.
// The remaining elements that were not cast are XORed conventionally.
// Of course a and dst must be the same length and the whole operation
// is performed in place.
// Only tested on AMD64.
func Xor(dst, a []byte) error {
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)

	for i := range castDst {
		castDst[i] ^= castA[i]
	}

	// deal with excess bytes which couldn't be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] ^= a[len(a)-j-1]
	}

	return nil
}

// And casts the first part of the byte slice (length divisible
// by 8) into uint64 and then performs AND on the slice of uint64.
// The remaining elements that were not cast are ANDed conventionally.
// Of course a and dst must be the same length and the whole operation
// is performed in place.
// Only tested on AMD64.
func And(dst, a []byte) error {
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)

	for i := range castDst {
		castDst[i] &= castA[i]
	}

	// deal with excess bytes which couldn't be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] &= a[len(a)-j-1]
	}

	return nil
}

// DoubleXor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs XOR on the slices of uint64
// (first with a and then with b). The remaining elements that were not
// cast are XORed conventionally. Of course a, b and dst must be the same
// length and the whole operation is performed in place.
// Only tested on AMD64.
func DoubleXor(dst, a, b []byte) error {
	if len(dst) != len(a) || len(dst) != len(b) {
		return ErrByteLengthMissMatch
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)
	castB := unsafeslice.Uint64SliceFromByteSlice(b)

	for i := range castDst {
		castDst[i] ^= castA[i]
		castDst[i] ^= castB[i]
	}

	// deal with excess bytes which couldn't be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] ^= a[len(a)-j-1]
		dst[len(dst)-j-1] ^= b[len(b)-j-1]
	}

	return nil
}

// AndXor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs AND on the slices of uint64
// (with a) and then performs XOR (with b). The remaining elements
// that were not cast are operated on conventionally. Of course a, b
// and dst must be the same length and the whole operation is
// performed in place.
// Only tested on AMD64.
func AndXor(dst, a, b []byte) error {
	if len(dst) != len(a) || len(dst) != len(b) {
		return ErrByteLengthMissMatch
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)
	castB := unsafeslice.Uint64SliceFromByteSlice(b)

	for i := range castDst {
		castDst[i] &= castA[i]
		castDst[i] ^= castB[i]
	}

	// deal with excess bytes which couldn't be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] &= a[len(a)-j-1]
		dst[len(dst)-j-1] ^= b[len(b)-j-1]
	}

	return nil
}

// ConcurrentBitOp performs an in-place bitwise operation, f, on each
// byte from a with dst if they are both the same length
func ConcurrentBitOp(f func([]byte, []byte) error, dst, a []byte) error {
	nworkers := runtime.GOMAXPROCS(0)
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
	if len(dst) < nworkers*16384 {
		return f(dst, a)
	}

	// determine number of blocks to split original matrix
	blockSize := len(dst) / nworkers

	// Run a worker pool
	var wg sync.WaitGroup
	wg.Add(nworkers)
	for w := 0; w < nworkers; w++ {
		w := w
		go func() {
			defer wg.Done()
			step := blockSize * w
			if w == nworkers-1 { // last block
				f(dst[step:], a[step:])
			} else {
				f(dst[step:step+blockSize], a[step:step+blockSize])
			}
		}()
	}

	wg.Wait()

	return nil
}

// ConcurrentDoubleBitOp performs an in-place double bitwise operation, f,
// on each byte from a with dst if they are both the same length
func ConcurrentDoubleBitOp(f func([]byte, []byte, []byte) error, dst, a, b []byte) error {
	nworkers := runtime.GOMAXPROCS(0)
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
	if len(dst) < nworkers*16384 {
		return f(dst, a, b)
	}

	// determine number of blocks to split original matrix
	blockSize := len(dst) / nworkers

	// Run a worker pool
	var wg sync.WaitGroup
	wg.Add(nworkers)
	for w := 0; w < nworkers; w++ {
		w := w
		go func() {
			defer wg.Done()
			step := blockSize * w
			if w == nworkers-1 { // last block
				f(dst[step:], a[step:], b[step:])
			} else {
				f(dst[step:step+blockSize], a[step:step+blockSize], b[step:step+blockSize])
			}
		}()
	}

	wg.Wait()

	return nil
}

// BitSetInByte returns true if bit i is set in a byte slice.
// It extracts bits from the least significant bit (i = 0) to the
// most significant bit (i = 7).
func BitSetInByte(b []byte, i int) bool {
	return b[i/8]&(1<<(i%8)) > 0
}

// SampleRandomBitMatrix allocates a 2D byte matrix of dimension row x col,
// and adds extra rows of 0s to have the number of rows be a multiple of 512,
// fills each entry in the byte matrix with pseudorandom byte values from a rand reader.
func SampleRandomBitMatrix(prng io.Reader, row, col int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, row)
	for row := range matrix {
		matrix[row] = make([]uint8, (col+PadTill512(col))/8)
	}
	// fill matrix
	for row := range matrix {
		if _, err := prng.Read(matrix[row]); err != nil {
			return nil, err
		}
	}

	return matrix, nil
}

// SampleRandom3DBitMatrix allocates a 3D byte matrix of dimension rows x cols with
// each column being another slice holding elems elements. Extra elements are added to
// each column to be a multiple of 512. Every slice is filled with pseudorandom bytes
// values from a rand reader.
func SampleRandom3DBitMatrix(prng io.Reader, rows, cols, elems int) ([][][]byte, error) {
	// instantiate matrix
	matrix := make([][][]byte, rows)
	for row := range matrix {
		matrix[row] = make([][]byte, cols)
		for col := range matrix[row] {
			matrix[row][col] = make([]byte, (elems+PadTill512(elems))/8)
			// fill
			if _, err := prng.Read(matrix[row][col]); err != nil {
				return nil, err
			}
		}
	}

	return matrix, nil

}

// PadTill512 returns the number of rows/columns to pad such that the number is a
// multiple of 512.
func PadTill512(m int) int {
	n := m % 512
	if n == 0 {
		return 0
	}

	return 512 - n
}
