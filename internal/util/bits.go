package util

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"

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
// This one iterates across the contiguous row and pulls
// values from the matrix
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
// This one iterates over the matrix and populates the
// contiguous row
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

// Convert 2D slices into a 1D slice where each row is
// contiguous
func Linearize2DMatrix(matrix [][]uint8) []uint8 {
	row := make([]uint8, len(matrix)*len(matrix[0]))

	for i := range matrix {
		copy(row[i*len(matrix[0]):], matrix[i])
	}

	return row
}

// Convert 1D slice into a 2D matrix with rows of desired width
func Reconstruct2DMatrix(row []uint8, width int) [][]uint8 {
	if len(row)%width != 0 {
		return nil
	}
	matrix := make([][]uint8, len(row)/width)
	for i := range matrix {
		matrix[i] = row[i*width : (i+1)*width]
	}
	return matrix
}

func swap(row []uint8, id1, id2 int) []uint8 {
	hold := row[id1]

	row[id1] = row[id2]
	row[id2] = hold

	return row
}

// where orig is original width and trans is transposed width
func ContiguousTranspose3(row []uint8, orig, trans int) []uint8 {
	transposed := make([]uint8, len(row))
	for i := range row {
		index := i / orig
		transposed[(i-(index*orig))*trans+index] = row[i]
	}
	return transposed
}

// The following two function have poor performance because the goroutines access a global slice which crosses cache lines
func ContiguousParallelTranspose(row, transposed []uint8, start, width, height int, wg *sync.WaitGroup) {
	defer wg.Done()
	index := start / width
	for i := start; i < start+width; i++ {
		transposed[(i-(index*width))*height+index] = row[i]
	}
}

func ContiguousParallelTranspose2(row []uint8, width, height int) []uint8 {
	var wg sync.WaitGroup
	transposed := make([]uint8, len(row))
	wg.Add(height)

	for i := 0; i < width*height; i += width {
		go func(i int) {
			defer wg.Done()
			index := i / width
			for j := i; j < i+width; j++ {
				transposed[(j-(index*width))*height+index] = row[j]
			}
		}(i)
	}
	wg.Wait()

	return transposed
}

func ContiguousParallelTranspose3(row []uint8, width, height int) []uint8 {
	length := width * height
	element := make(chan uint8, length)
	location := make(chan int, length)

	transposed := make([]uint8, length)

	for i := 0; i < length; i++ {
		index := i / width
		go func(i int, e chan<- uint8, l chan<- int) {
			element <- row[i]
			location <- (i-(index*width))*height + index
		}(i, element, location)
	}

	// pull from channel to populate new list
	go func() {
		for j := 0; j < length; j++ {
			transposed[<-location] = <-element
		}
	}()
	/*
		close(location)
		close(element)
	*/
	return transposed
}

/*
// width is width of transposed matrix
func ContiguousTranspose4(row []uint8, width int) []uint8 {
	transposed := make([]uint8, len(row))
	j := 0
	for i := range row {
		transposed[i%width+j] = row[i]
		if i%width == 0 {
			j = 0
		} else {
			j += 1
		}
	}
	return transposed
}
*/
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
/*
// RecTranspose
func TransposeRec(matrix [][]uint8) [][][]uint8 {
	m := len(matrix)
	k := len(matrix[0])
	// cacheLimit/2 + 1 represents the the maximum size
	// matrix with which you'll need to iterate using
	// the naive algorithm
	cacheLimit := 4

	// create k x m transposed matrix to store values
	transposed := make([][]uint8, k)
	for r := range transposed {
		transposed[r] = make([]uint8, m)
	}

	mb := m
	kb := k
	// recursively divide the matrices
	for mb > cacheLimit || kb > cacheLimit {
		if mb > kb {
			mb /= 2
		} else {
			kb /= 2
		}
	}

	mt := m
	kt := k
	// populate the transposed matrix with divided blocks
	for i := 0; mt < len(matrix) || kt < len(matrix[0]); i++ {
		if kb*(i+1) > k {
			// finish the unusual block size
			for _, row := range transposed[kb*i:] {

			}
		} else {
			for _, row := range transposed[kb*i:kb*(i+1)] {
				row = 4
		}
		}
	}


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

// Extract a contiguous slice of bytes from slices of BitSets
func BitSetsToByteSlice(bsets []*bitset.BitSet) []byte {
	bLen := len(bsets[0].Bytes()) * 8
	b := make([]byte, len(bsets)*bLen)

	for i, x := range bsets {
		for j, y := range x.Bytes() {
			binary.LittleEndian.PutUint64(b[i*bLen+j*8:], y)
		}
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

// Convert slice of bytes (representing a contiguous matrix)
// to slices of BitSets
func ByteSliceToBitsets(b []byte, width int) []*bitset.BitSet {
	bsets := make([]*bitset.BitSet, len(b)/width)

	for i := range bsets {
		bsets[i] = BytesToBitSet(b[i*width : (i+1)*width])
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
