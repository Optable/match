package util

import (
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

// InplaceXorBytes XORS each byte from a with dst in place
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

// InplaceDoubleXorBytes XORS each byte from a with dst in place
// followed by XORing each byte from b with dst in place if a, b
// and dst are all the same length
func InPlaceDoubleXorBytes(dst, a, b []byte) error {
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := 0; i < n; i++ {
		dst[i] ^= a[i]
		dst[i] ^= b[i]
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

// ConcurrentInPlaceDoubleXorBytes XORS each byte from a with dst in place
// followed by XORing each byte from b with dst in place if a, b and dst
// are all the same length
func ConcurrentInPlaceDoubleXorBytes(dst, a, b []byte) error {
	const blockSize int = 16384 // half of what L2 cache can hold
	nworkers := runtime.GOMAXPROCS(0)
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
	if n < blockSize {
		return InPlaceDoubleXorBytes(dst, a, b)
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
						dst[i] ^= b[i]
					}
				} else {
					for i := step; i < step+blockSize; i++ {
						dst[i] ^= a[i]
						dst[i] ^= b[i]
					}
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

// AndBytes returns the binary AND of each byte in a and b
// and returns dst if a and b are the same length
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

// InPlaceAndXorBytes performs the binary AND of each
// byte in a and dst followed by XORing each byte in b and dst
// if a, b and dst are the same length.
func InPlaceAndXorBytes(dst, a, b []byte) error {
	n := len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := range dst {
		dst[i] &= a[i]
		dst[i] ^= b[i]
	}

	return nil
}

// ConcurrentInPlaceAndXorBytes performs the binary AND of each
// byte in a and dst followed by XORing each byte in b and dst
// if a, b and dst are the same length.
func ConcurrentInPlaceAndXorBytes(dst, a, b []byte) error {
	const blockSize int = 16384 // half of what L2 cache can hold
	nworkers := runtime.GOMAXPROCS(0)
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
	if n < blockSize {
		return InPlaceAndXorBytes(dst, a, b)
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
						dst[i] ^= b[i]
					}
				} else {
					for i := step; i < step+blockSize; i++ {
						dst[i] &= a[i]
						dst[i] ^= b[i]
					}
				}
			}
		}()
	}

	wg.Wait()

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
// It extracts bits from the least significant bit (i = 0) to the
// most significant bit (i = 7).
func TestBitSetInByte(b []byte, i int) byte {
	if b[i/8]&(1<<(i%8)) == 0 {
		return 0
	}
	return 1
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
