package util

import (
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/alecthomas/unsafeslice"
)

var ErrByteLengthMissMatch = fmt.Errorf("provided bytes do not have the same length for bit operations")

// Xor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs XOR on the slices of uint64.
// The remaining elements that were not cast are XORed conventionally.
// Of course a and dst must be the same length and the whole operation
// is performed in place.
func Xor(dst, a []byte) error {
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)

	for i := range castDst {
		castDst[i] ^= castA[i]
	}

	copy(dst, unsafeslice.ByteSliceFromUint64Slice(castDst))

	// deal with excess bytes which couldn't be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] ^= a[len(a)-j-1]
	}

	return nil
}

// And casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs AND on the slices of uint64.
// The remaining elements that were not cast are ANDed conventionally.
// Of course a and dst must be the same length and the whole operation
// is performed in place.
func And(dst, a []byte) error {
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)

	for i := range castDst {
		castDst[i] &= castA[i]
	}

	copy(dst, unsafeslice.ByteSliceFromUint64Slice(castDst))

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
// cast are XORed conventionally. Of course a and dst must be the same
// length and the whole operation is performed in place.
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

	copy(dst, unsafeslice.ByteSliceFromUint64Slice(castDst))

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
// that were not cast are operated conventionally. Of course a and dst
// must be the same length and the whole operation is performed in place.
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

	copy(dst, unsafeslice.ByteSliceFromUint64Slice(castDst))

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
	const blockSize int = 16384 // half of what L2 cache can hold
	nworkers := runtime.GOMAXPROCS(0)
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
	if len(dst) < blockSize {
		return f(dst, a)
	}

	// determine number of blocks to split original matrix
	nblks := len(dst) / blockSize
	if len(dst)%blockSize != 0 {
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
			var step int
			for blk := range ch {
				step = blockSize * blk
				if blk == nblks-1 { // last block
					f(dst[step:], a[step:])
				} else {
					f(dst[step:step+blockSize], a[step:step+blockSize])
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

// ConcurrentDoubleBitOp performs an in-place double bitwise operation, f,
// on each byte from a with dst if they are both the same length
func ConcurrentDoubleBitOp(f func([]byte, []byte, []byte) error, dst, a, b []byte) error {
	const blockSize int = 16384 // half of what L2 cache can hold
	nworkers := runtime.GOMAXPROCS(0)
	if len(dst) != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
	if len(dst) < blockSize {
		return f(dst, a, b)
	}

	// determine number of blocks to split original matrix
	nblks := len(dst) / blockSize
	if len(dst)%blockSize != 0 {
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
			var step int
			for blk := range ch {
				step = blockSize * blk
				if blk == nblks-1 { // last block
					f(dst[step:], a[step:], b[step:])
				} else {
					f(dst[step:step+blockSize], a[step:step+blockSize], b[step:step+blockSize])
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

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

// InPlaceXorBytes XORS each byte from a with dst in place
// if a and dst are the same length
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

// ConcurrentInPlaceXorBytes XORS each byte from a with dst in place
// if a and dst are the same length
func ConcurrentInPlaceXorBytes(dst, a []byte) error {
	const blockSize int = 16384 // half of what L2 cache can hold
	nworkers := runtime.GOMAXPROCS(0)
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
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

// InPlaceAndBytes performs the binary AND of each byte in a
// and dst in place if a and dst are the same length.
func InPlaceAndBytes(dst, a []byte) error {
	n := len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := range dst {
		dst[i] = dst[i] & a[i]
	}

	return nil
}

// ConcurrentInPlaceAndBytes performs the binary AND of each
// byte in a and dst if a and dst are the same length.
func ConcurrentInPlaceAndBytes(dst, a []byte) error {
	const blockSize int = 16384 // half of what L2 cache can hold
	nworkers := runtime.GOMAXPROCS(0)
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
	if n < blockSize {
		return InPlaceAndBytes(dst, a)
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
						dst[i] &= a[i]
					}
				} else {
					for i := step; i < step+blockSize; i++ {
						dst[i] &= a[i]
					}
				}
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

// Transpose returns the transpose of a 2D slices of bytes
// from (m x k) to (k x m) by naively swapping.
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

// TransposeByteMatrix performs a concurrent cache-oblivious transpose on a byte matrix by first
// converting from bytes to uint64 (and padding as needed), performing the transpose on the uint64
// matrix and then converting back to bytes.
// pad is number of rows containing 0s which will be added to end of matrix.
// Assume each row contains 64 bytes (512 bits).
func TransposeByteMatrix(b [][]byte) (tr [][]byte) {
	pad := PadTill512(len(b))
	for i := 0; i < pad; i++ {
		b = append(b, make([]byte, 64))
	}
	return ConcurrentTranspose(b, runtime.NumCPU())
}
