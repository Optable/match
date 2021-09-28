package util

import (
	"fmt"
	"io"
	"sync"
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
		dst[i] &= a[i]
	}

	return nil
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
func SampleRandomBitMatrix(prng io.Reader, m, k int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, m)
	for row := range matrix {
		matrix[row] = make([]uint8, k)
	}

	for row := range matrix {
		if err := SampleBitSlice(prng, matrix[row]); err != nil {
			return nil, err
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

// The major failing of my last attempt at concurrent transposition
// was that each goroutine (and there were far too many) was accessing
// the same shared array. This meant that the cache on each core had to
// be constantly updated as each coroutine updated their cache-local
// version. Instead this version splits everything by column. Each
// goroutine reads from the same shared matrix, but since nothing is
// is changed, the local cache shouldn't need an update. Then each
// goroutine sends the transposed row back to a channel in an ordered set.
// Once all transpositions are done, rows are recombined into a 2D matrix.
func ConcurrentColumnarTranspose(matrix [][]uint8) [][]uint8 {
	var wg sync.WaitGroup
	m := len(matrix)
	n := len(matrix[0])
	tr := make([][]uint8, n)

	// the optimal number of goroutines will likely vary due to
	// hardware and array size
	//nThreads := n // one thread per column (likely only efficient for huge matrix)
	// nThreads := runtime.NumCPU()
	// nThreads := runtime.NumCPU()*2
	nThreads := 12
	// add to quick check to ensure there are not more threads than columns
	if n < nThreads {
		nThreads = n
	}

	// number of columns for which each goroutine is responsible
	nColumns := n / nThreads

	// create ordered channels to store values from goroutines
	// each channel is buffered to store the number of desired rows
	channels := make([]chan []uint8, nThreads)
	for i := 0; i < nThreads-1; i++ {
		channels[i] = make(chan []uint8, nColumns)
	}
	// last one may have excess columns
	channels[nThreads-1] = make(chan []uint8, nColumns+n%nThreads)

	// goroutine
	//wg.Add(nThreads)
	for i := 0; i < nThreads; i++ {
		wg.Add(1)
		go func(i int) {
			//	fmt.Println("goroutine", i, "created")
			defer wg.Done()
			// we need to handle excess columns which don't evenly divide among
			// number of threads -> in this case, I just add to the last goroutine
			// perhaps a more sophisticated division of labor would be more efficient
			var extraColumns int
			if i == nThreads-1 {
				extraColumns = n % nThreads
			}

			for c := 0; c < (nColumns + extraColumns); c++ {
				row := make([]uint8, m)
				for r := 0; r < m; r++ {
					row[r] = matrix[r][(i*nColumns)+c]
				}

				channels[i] <- row
			}

			close(channels[i])
		}(i)
	}

	// Wait until all goroutines have finished
	wg.Wait()

	// Reconstruct a transposed matrix from the channels
	for i, channel := range channels {
		var j int
		for row := range channel {
			tr[(i*nColumns)+j] = row
			j++
		}
	}

	return tr
}
