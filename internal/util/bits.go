package util

import (
	"crypto/rand"
	"fmt"
	"runtime"
	"sync"

	"github.com/alecthomas/unsafeslice"
)

var ErrByteLengthMissMatch = fmt.Errorf("provided bytes do not have the same length for bit operations")

// Xor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs XOR on the slices of uint64.
// The excess elements that could not be cast are XORed conventionally.
// The whole operation is performed in place. Panic if a and dst do
// not have the same length.
// Only tested on x86-64.
func Xor(dst, a []byte) {
	if len(dst) != len(a) {
		panic(ErrByteLengthMissMatch)
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)

	for i := range castDst {
		castDst[i] ^= castA[i]
	}

	// deal with excess bytes which could not be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] ^= a[len(a)-j-1]
	}
}

// And casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs AND on the slices of uint64.
// The excess elements that could not be cast are ANDed conventionally.
// The whole operation is performed in place. Panic if a and dst do
// not have the same length.
// Only tested on x86-64.
func And(dst, a []byte) {
	if len(dst) != len(a) {
		panic(ErrByteLengthMissMatch)
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)

	for i := range castDst {
		castDst[i] &= castA[i]
	}

	// deal with excess bytes which could not be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] &= a[len(a)-j-1]
	}
}

// DoubleXor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs XOR on the slices of uint64
// (first with a and then with b). The excess elements that could not
// be cast are XORed conventionally. The whole operation is performed
// in place. Panic if a, b and dst do not have the same length.
// Only tested on x86-64.
func DoubleXor(dst, a, b []byte) {
	if len(dst) != len(a) || len(dst) != len(b) {
		panic(ErrByteLengthMissMatch)
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)
	castB := unsafeslice.Uint64SliceFromByteSlice(b)

	for i := range castDst {
		castDst[i] ^= castA[i]
		castDst[i] ^= castB[i]
	}

	// deal with excess bytes which could not be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] ^= a[len(a)-j-1]
		dst[len(dst)-j-1] ^= b[len(b)-j-1]
	}
}

// AndXor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs AND on the slices of uint64
// (with a) and then performs XOR (with b). The excess elements
// that could not be cast are operated on conventionally. The whole
// operation is performed in place. Panic if a, b and dst do not
// have the same length.
// Only tested on x86-64.
func AndXor(dst, a, b []byte) {
	if len(dst) != len(a) || len(dst) != len(b) {
		panic(ErrByteLengthMissMatch)
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)
	castB := unsafeslice.Uint64SliceFromByteSlice(b)

	for i := range castDst {
		castDst[i] &= castA[i]
		castDst[i] ^= castB[i]
	}

	// deal with excess bytes which could not be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] &= a[len(a)-j-1]
		dst[len(dst)-j-1] ^= b[len(b)-j-1]
	}
}

// ConcurrentBitOp performs an in-place bitwise operation, f, on each
// byte from a with dst if they are both the same length.
func ConcurrentBitOp(f func([]byte, []byte), dst, a []byte) {
	nworkers := runtime.GOMAXPROCS(0)

	// no need to split into goroutines
	if len(dst) < nworkers*16384 {
		f(dst, a)
	} else {

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
	}
}

// ConcurrentDoubleBitOp performs an in-place double bitwise operation, f,
// on each byte from a with dst if they are both the same length
func ConcurrentDoubleBitOp(f func([]byte, []byte, []byte), dst, a, b []byte) {
	nworkers := runtime.GOMAXPROCS(0)

	// no need to split into goroutines
	if len(dst) < nworkers*16384 {
		f(dst, a, b)
	} else {

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
	}
}

// IsBitSet returns true if bit i is set in a byte slice.
// It extracts bits from the least significant bit (i = 0) to the
// most significant bit (i = 7).
func IsBitSet(b []byte, i int) bool {
	return b[i/8]&(1<<(i%8)) > 0
}

// BitExtract returns the ith bit in b
func BitExtract(b []byte, i int) byte {
	if IsBitSet(b, i) {
		return 1
	}

	return 0
}

// SampleRandomBitMatrix allocates a 2D byte matrix of dimension row x col,
// and adds extra rows of 0s to have the number of rows be a multiple of 512,
// fills each entry in the byte matrix with pseudorandom byte values from a rand reader.
func SampleRandomBitMatrix(row, col int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, row)
	for row := range matrix {
		matrix[row] = make([]uint8, (col+Pad(col, 512))/8)
	}
	// fill matrix
	for row := range matrix {
		if _, err := rand.Read(matrix[row]); err != nil {
			return nil, err
		}
	}

	return matrix, nil
}

// Pad returns the total padded length such that n is padded to a multiple of
// multiple.
func Pad(n, multiple int) int {
	p := n % multiple
	if p == 0 {
		return n
	}

	return n + (multiple - p)
}

// PadBitMap returns the total padded length such that n is padded to a multiple of
// multiple bytes to fit in a bitmap.
func PadBitMap(n, multiple int) int {
	return Pad(n, multiple) / 8
}
