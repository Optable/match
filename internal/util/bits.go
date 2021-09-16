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

func XorByteRowCol(a []byte, b [][]byte, index int) (dst []byte, err error) {
	n := len(b)
	if n != len(a) {
		return nil, ErrByteLengthMissMatch
	}

	dst = make([]byte, n)

	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i][index]
	}

	return
}

func XorByteRowColInPlace(a []byte, b [][]byte, index int) (err error) {
	n := len(b)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := 0; i < n; i++ {
		a[i] = a[i] ^ b[i][index]
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

	if a == 0 {
		return dst
	}

	return b
}

func AndBitSet(a bool, b *bitset.BitSet) *bitset.BitSet {
	// Create a bitset of zeros with the appropriate length
	// You could actually instantiate a bitset with length 0
	// and the bitwise operations would still work but you
	// can end up with a resultant bitset with zero length
	// because you've performed a bitset operation with a null
	// bitset
	aSet := bitset.New(b.Len())
	if a {
		return b
	}

	return aSet
}

// This is a more efficient method than that listed above
func InPlaceAndBitSet(a bool, b *bitset.BitSet) {
	if !a {
		b.ClearAll()
	}
}

func GetCol(matrix [][]uint8, index int) []uint8 {
	col := make([]uint8, len(matrix))
	for i := range matrix {
		col[i] = matrix[i][index]
	}
	return col
}

func GetBitSetCol(matrix []*bitset.BitSet, index uint) *bitset.BitSet {
	col := bitset.New(uint(len(matrix)))
	for i, row := range matrix {
		if row.Test(index) {
			col.Set(uint(i))
		}
	}

	return col
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

// This transpose iterates in reserve order and is less efficient
/*
func Transpose(matrix [][]uint8) [][]uint8 {
	n := len(matrix)
	tr := make([][]uint8, len(matrix[0]))

	for row := range tr {
		tr[row] = make([]uint8, n)
	}

	for col := range tr[0] {
		for row := range tr {
			tr[row][col] = matrix[col][row]

		}
	}
	return tr
}
*/

// This tranpose function returns the transpose of a 2D slice of uint8
// Due to limitations of golang, appending a series of transposed matrix
// chunks is easiest (maybe not most efficient) when each chunk was
// produced using a rectangular matrix of some width but height equal to
// the original matrix. This means that reconstructing the full matrix at
// the end is simply a matter of appending the chunks directly.
/*
func VerticalTranspose(matrix [][]uint8, chunkWidth int) [][]uint8 {
	height := len(matrix)
	width := len(matrix[0])
	var numChunks int
	if width % chunkWidth != 0 {
		numChunks = width/chunkWidth + 1
	} else {
		numChunks = width/chunkWidth
	}
	chunks := make([][][]uint8, numChunks)

	for i := 0, i < numChunks, i++ {
		chunk[i] := make([][]uint8, chunkWidth)
		for j :=

	}
}
*/
// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
// This one iterates across the contiguous row and pulls
// values from the matrix
func contiguousTranspose2(matrix [][]uint8) [][]uint8 {
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

/*
// Cache-oblivious transpose
// Recursive matrix transposition that
func CacheObliviousTranspose(matrix, transposed [][]uint8, minBlock, blockHeight, blockWidth, indexHeight, indexWidth uint) {
	if blockHeight < minBlock {
		for row := indexHeight; row < indexHeight+blockHeight; row++ {
			for col := indexWidth; col < indexWidth+blockWidth; col++ {
				transposed[col][row] = matrix[row][col]
			}
		}
	} else {
		// subdivide by long side
		if blockHeight > blockWidth {
			CacheObliviousTranspose(matrix, transposed, minBlock, blockHeight/2, blockWidth, indexHeight, indexWidth)
			CacheObliviousTranspose(matrix, transposed, minBlock, blockHeight/2, blockWidth, indexHeight+blockHeight/2, indexWidth)
		} else {
			CacheObliviousTranspose(matrix, transposed, minBlock, blockHeight, blockWidth/2, indexHeight, indexWidth)
			CacheObliviousTranspose(matrix, transposed, minBlock, blockHeight, blockWidth/2, indexHeight, indexWidth+blockWidth/2)
		}
	}
}
*/
// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
// This is MORE efficient that the other version
// This one iterates over the matrix and populates the
// contiguous row
func ContiguousTranspose(matrix [][]uint8) [][]uint8 {
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

// Transpose returns the transpose of a 3D slices of uint8
// from (m x k) to (k x m)
// This one iterates over the matrix and populates the
// contiguous row
func ContiguousTranspose3D(matrix [][][]uint8) [][][]uint8 {
	depth := len(matrix)
	height := len(matrix[0])
	width := len(matrix[0][0])
	tr := make([][][]uint8, width)
	longRow := make([]uint8, depth*height*width)

	for i := 0; i < depth; i++ {
		for j := 0; j < height; j++ {
			for k := 0; k < width; k++ {
				longRow[k*(height*depth)+j*depth+i] = matrix[i][j][k]
			}
		}
	}

	for i := range tr {
		tr[i] = make([][]uint8, height)
		for j := range tr[i] {
			pindex := i * (height * depth)
			tr[i][j] = longRow[pindex+j*depth : pindex+(j+1)*depth]
		}
	}

	return tr
}

// Convert 2D slices into a 1D slice where each row is
// contiguous
func linearize2DMatrix(matrix [][]uint8) []uint8 {
	row := make([]uint8, len(matrix)*len(matrix[0]))

	for i := range matrix {
		copy(row[i*len(matrix[0]):], matrix[i])
	}

	return row
}

// This is basically the same as the standard transpose
func ColumnarTranspose(matrix [][]uint8) [][]uint8 {
	m := len(matrix)
	n := len(matrix[0])
	tr := make([][]uint8, n)

	for i := 0; i < n; i++ {
		tr[i] = make([]uint8, m)
		for j := 0; j < m; j++ {
			tr[i][j] = matrix[j][i]
		}
	}

	return tr
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

// Convert 1D slice into a 2D matrix with rows of desired width
func reconstruct2DMatrix(row []uint8, width int) [][]uint8 {
	if len(row)%width != 0 {
		return nil
	}
	matrix := make([][]uint8, len(row)/width)
	for i := range matrix {
		matrix[i] = row[i*width : (i+1)*width]
	}
	return matrix
}

// where orig is original width and trans is transposed width
func contiguousTranspose3(row []uint8, orig, trans int) []uint8 {
	transposed := make([]uint8, len(row))
	for i := range row {
		index := i / orig
		transposed[(i-(index*orig))*trans+index] = row[i]
	}
	return transposed
}

// The following two function have poor performance because the goroutines access a global slice which crosses cache lines
func contiguousParallelTranspose(row, transposed []uint8, start, width, height int, wg *sync.WaitGroup) {
	defer wg.Done()
	index := start / width
	for i := start; i < start+width; i++ {
		transposed[(i-(index*width))*height+index] = row[i]
	}
}

func contiguousParallelTranspose2(row []uint8, width, height int) []uint8 {
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

func contiguousParallelTranspose3(row []uint8, width, height int) []uint8 {
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
	// read up to len(b) + 1 pseudorandom bits
	t := make([]byte, len(b)/8+1)
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

// Extract a slice of bytes from a BitSet
func BitSetToBytes(bset *bitset.BitSet) []byte {
	b := make([]byte, len(bset.Bytes())*8)

	for i, x := range bset.Bytes() {
		binary.LittleEndian.PutUint64(b[i*8:], x)
	}

	return b
}

// Extract a slice of bytes (each holding a single bit)
// from a BitSet
func BitSetToBits(bset *bitset.BitSet) []byte {
	b := make([]byte, bset.Len())

	for i := range b {
		if bset.Test(uint(i)) {
			b[i] = 1
		}
	}

	return b
}

// Extract a matrix of bytes from slices of BitSets
func bitSetsToByteMatrix(bsets []*bitset.BitSet) [][]byte {
	b := make([][]byte, len(bsets))

	for i, x := range bsets {
		b[i] = BitSetToBytes(x)
	}

	return b
}

// Extract a matrix of bytes (each holding a single bit)
// from slices of BitSets
func BitSetsToBitMatrix(bsets []*bitset.BitSet) [][]byte {
	b := make([][]byte, len(bsets))

	for i, x := range bsets {
		b[i] = BitSetToBits(x)
	}

	return b
}

// Convert 2D bitset matrix to a matrix containing the uint64 values representing the bitsets
func BitSetsToUints(bsets []*bitset.BitSet) [][]uint64 {
	b := make([][]uint64, len(bsets))

	for i, x := range bsets {
		b[i] = x.Bytes()
	}

	return b
}

// Convert 2D matrix of uint64 into a 2D bitset array
func UintsToBitSets(usets [][]uint64) []*bitset.BitSet {
	b := make([]*bitset.BitSet, len(usets))

	for i, x := range usets {
		b[i] = bitset.From(x)
	}

	return b
}

func BitSetsToBitMatrix3D(bsets [][]*bitset.BitSet) [][][]byte {
	b := make([][][]byte, len(bsets))

	for i, x := range bsets {
		b[i] = BitSetsToBitMatrix(x)
	}

	return b
}

// Extract a contiguous slice of bytes from slices of BitSets
func bitSetsToByteSlice(bsets []*bitset.BitSet) []byte {
	bLen := len(bsets[0].Bytes()) * 8
	b := make([]byte, len(bsets)*bLen)

	for i, x := range bsets {
		for j, y := range x.Bytes() {
			binary.LittleEndian.PutUint64(b[i*bLen+j*8:], y)
		}
	}
	return b
}

// Extract a contiguous slice of bytes (each holding a single bit)
// from slices of BitSets
func bitSetsToBitSlice(bsets []*bitset.BitSet) []byte {
	bLen := int(bsets[0].Len())
	b := make([]byte, len(bsets)*bLen)

	for i := range bsets {
		for j := 0; j < bLen; j++ {
			if bsets[i].Test(uint(j)) {
				b[i*bLen+j] = 1
			}
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

// Convert slice of bytes (each holding a single bit)
// to Bitset (note that the BitSet structure will add
// additional zeros since it is truly a uint64)
func BitsToBitSet(b []byte) *bitset.BitSet {
	// expand byte slice to a multiple of 8
	/*
		var x int
		if len(b)%8 != 0 {
			x = 8 - (len(b) % 8)
		}

		b = append(b, make([]byte, x)...)
	*/
	bset := bitset.New(uint(len(b)))

	for i, x := range b {
		if x == 1 {
			bset.Set(uint(i))
		}
	}

	return bset
}

// Convert matrix of bytes to slices of BitSets
func byteMatrixToBitSets(b [][]byte) []*bitset.BitSet {
	bsets := make([]*bitset.BitSet, len(b))

	for i, x := range b {
		bsets[i] = BytesToBitSet(x)
	}

	return bsets
}

// Convert matrix of bytes (each holding a single bit)
// to slices of BitSets
func BitMatrixToBitSets(b [][]byte) []*bitset.BitSet {
	bsets := make([]*bitset.BitSet, len(b))

	for i, x := range b {
		bsets[i] = BitsToBitSet(x)
	}

	return bsets
}

// Convert matrix of bytes (each holding a single bit)
// to slices of BitSets
func BitMatrixToBitSets3D(b [][][]byte) [][]*bitset.BitSet {
	bsets := make([][]*bitset.BitSet, len(b))

	for i, x := range b {
		bsets[i] = BitMatrixToBitSets(x)
	}

	return bsets
}

// Convert slice of bytes (representing a contiguous matrix)
// to slices of BitSets
func byteSliceToBitSets(b []byte, width int) []*bitset.BitSet {
	bsets := make([]*bitset.BitSet, len(b)/width)

	for i := range bsets {
		bsets[i] = BytesToBitSet(b[i*width : (i+1)*width])
	}
	return bsets
}

// Convert slices of bytes (each holding a single bit and
// the whole slice representing a contiguous matrix) to
// slices of BitSets
func BitSliceToBitSets(b []byte, width int) []*bitset.BitSet {
	bsets := make([]*bitset.BitSet, len(b)/width)

	for i := range bsets {
		bsets[i] = BitsToBitSet(b[i*width : (i+1)*width])
	}
	return bsets
}

// from: https://books.google.ca/books?id=VicPJYM0I5QC&pg=PA145&lpg=PA145&dq=recursive+bit+matrix+transpose&source=bl&ots=2p1OPRur0n&sig=ACfU3U1fTJLM78DAHQwocR2YbyRYgF9DEA&hl=en&sa=X&ved=2ahUKEwiU-oTt4v7yAhXrYt8KHVTqDWMQ6AF6BAgWEAM#v=onepage&q=recursive%20bit%20matrix%20transpose&f=false
// modified bitset transpose for a slice of uint64 representing a 64x64 matrix of bits
/*
[00 01 02 . . . 63]
is representing this:
[
	00
	01
	02
	.
	.
	.
	63
]
which in binary is:
[
	00-63 00-62 00-61 . . . 00-00
	01-63 01-62 01-61 . . . 01-00
	02-63 02-62 02-61 . . . 02-00
	.
	.
	.
	63-63 63-62 63-61 . . . 63-00
]
this is the structure we are transposing
*/
func Transpose64(A []uint64) {
	var width, k int = 32, 0
	var mask, t uint64 = 0x00000000FFFFFFFF, 0

	for width != 0 {
		for k = 0; k < 64; k = ((k | width) + 1) &^ width {
			t = (A[k] ^ (A[k|width] >> width)) & mask
			A[k] = A[k] ^ t
			A[k|width] = A[k|width] ^ (t << width)
		}

		width >>= 1
		mask ^= mask << width
	}
}

func swap(a, b int, array []uint64, mask uint64, width int) {
	t := (array[a] ^ (array[b] >> width)) & mask
	array[a] = array[a] ^ t
	array[b] = array[b] ^ (t << width)
}

func UnrolledTranspose64(array []uint64) {
	// 32x32 swap
	var mask uint64 = 0x00000000FFFFFFFF
	var width int = 32
	swap(0, 32, array, mask, width)
	swap(1, 33, array, mask, width)
	swap(2, 34, array, mask, width)
	swap(3, 35, array, mask, width)
	swap(4, 36, array, mask, width)
	swap(5, 37, array, mask, width)
	swap(6, 38, array, mask, width)
	swap(7, 39, array, mask, width)
	swap(8, 40, array, mask, width)
	swap(9, 41, array, mask, width)
	swap(10, 42, array, mask, width)
	swap(11, 43, array, mask, width)
	swap(12, 44, array, mask, width)
	swap(13, 45, array, mask, width)
	swap(14, 46, array, mask, width)
	swap(15, 47, array, mask, width)
	swap(16, 48, array, mask, width)
	swap(17, 49, array, mask, width)
	swap(18, 50, array, mask, width)
	swap(19, 51, array, mask, width)
	swap(20, 52, array, mask, width)
	swap(21, 53, array, mask, width)
	swap(22, 54, array, mask, width)
	swap(23, 55, array, mask, width)
	swap(24, 56, array, mask, width)
	swap(25, 57, array, mask, width)
	swap(26, 58, array, mask, width)
	swap(27, 59, array, mask, width)
	swap(28, 60, array, mask, width)
	swap(29, 61, array, mask, width)
	swap(30, 62, array, mask, width)
	swap(31, 63, array, mask, width)
	// 16x16 swap
	mask = 0x0000FFFF0000FFFF
	width = 16
	swap(0, 16, array, mask, width)
	swap(1, 17, array, mask, width)
	swap(2, 18, array, mask, width)
	swap(3, 19, array, mask, width)
	swap(4, 20, array, mask, width)
	swap(5, 21, array, mask, width)
	swap(6, 22, array, mask, width)
	swap(7, 23, array, mask, width)
	swap(8, 24, array, mask, width)
	swap(9, 25, array, mask, width)
	swap(10, 26, array, mask, width)
	swap(11, 27, array, mask, width)
	swap(12, 28, array, mask, width)
	swap(13, 29, array, mask, width)
	swap(14, 30, array, mask, width)
	swap(15, 31, array, mask, width)

	swap(32, 48, array, mask, width)
	swap(33, 49, array, mask, width)
	swap(34, 50, array, mask, width)
	swap(35, 51, array, mask, width)
	swap(36, 52, array, mask, width)
	swap(37, 53, array, mask, width)
	swap(38, 54, array, mask, width)
	swap(39, 55, array, mask, width)
	swap(40, 56, array, mask, width)
	swap(41, 57, array, mask, width)
	swap(42, 58, array, mask, width)
	swap(43, 59, array, mask, width)
	swap(44, 60, array, mask, width)
	swap(45, 61, array, mask, width)
	swap(46, 62, array, mask, width)
	swap(47, 63, array, mask, width)
	// 8x8 swap
	mask = 0x00FF00FF00FF00FF
	width = 8
	swap(0, 8, array, mask, width)
	swap(1, 9, array, mask, width)
	swap(2, 10, array, mask, width)
	swap(3, 11, array, mask, width)
	swap(4, 12, array, mask, width)
	swap(5, 13, array, mask, width)
	swap(6, 14, array, mask, width)
	swap(7, 15, array, mask, width)

	swap(16, 24, array, mask, width)
	swap(17, 25, array, mask, width)
	swap(18, 26, array, mask, width)
	swap(19, 27, array, mask, width)
	swap(20, 28, array, mask, width)
	swap(21, 29, array, mask, width)
	swap(22, 30, array, mask, width)
	swap(23, 31, array, mask, width)

	swap(32, 40, array, mask, width)
	swap(33, 41, array, mask, width)
	swap(34, 42, array, mask, width)
	swap(35, 43, array, mask, width)
	swap(36, 44, array, mask, width)
	swap(37, 45, array, mask, width)
	swap(38, 46, array, mask, width)
	swap(39, 47, array, mask, width)

	swap(48, 56, array, mask, width)
	swap(49, 57, array, mask, width)
	swap(50, 58, array, mask, width)
	swap(51, 59, array, mask, width)
	swap(52, 60, array, mask, width)
	swap(53, 61, array, mask, width)
	swap(54, 62, array, mask, width)
	swap(55, 63, array, mask, width)
	// 4x4 swap
	mask = 0x0F0F0F0F0F0F0F0F
	width = 4
	swap(0, 4, array, mask, width)
	swap(1, 5, array, mask, width)
	swap(2, 6, array, mask, width)
	swap(3, 7, array, mask, width)

	swap(8, 12, array, mask, width)
	swap(9, 13, array, mask, width)
	swap(10, 14, array, mask, width)
	swap(11, 15, array, mask, width)

	swap(16, 20, array, mask, width)
	swap(17, 21, array, mask, width)
	swap(18, 22, array, mask, width)
	swap(19, 23, array, mask, width)

	swap(24, 28, array, mask, width)
	swap(25, 29, array, mask, width)
	swap(26, 30, array, mask, width)
	swap(27, 31, array, mask, width)

	swap(32, 36, array, mask, width)
	swap(33, 37, array, mask, width)
	swap(34, 38, array, mask, width)
	swap(35, 39, array, mask, width)

	swap(40, 44, array, mask, width)
	swap(41, 45, array, mask, width)
	swap(42, 46, array, mask, width)
	swap(43, 47, array, mask, width)

	swap(48, 52, array, mask, width)
	swap(49, 53, array, mask, width)
	swap(50, 54, array, mask, width)
	swap(51, 55, array, mask, width)

	swap(56, 60, array, mask, width)
	swap(57, 61, array, mask, width)
	swap(58, 62, array, mask, width)
	swap(59, 63, array, mask, width)
	// 2x2 swap
	mask = 0x3333333333333333
	width = 2
	swap(0, 2, array, mask, width)
	swap(1, 3, array, mask, width)

	swap(4, 6, array, mask, width)
	swap(5, 7, array, mask, width)

	swap(8, 10, array, mask, width)
	swap(9, 11, array, mask, width)

	swap(12, 14, array, mask, width)
	swap(13, 15, array, mask, width)

	swap(16, 18, array, mask, width)
	swap(17, 19, array, mask, width)

	swap(20, 22, array, mask, width)
	swap(21, 23, array, mask, width)

	swap(24, 26, array, mask, width)
	swap(25, 27, array, mask, width)

	swap(28, 30, array, mask, width)
	swap(29, 31, array, mask, width)

	swap(32, 34, array, mask, width)
	swap(33, 35, array, mask, width)

	swap(36, 38, array, mask, width)
	swap(37, 39, array, mask, width)

	swap(40, 42, array, mask, width)
	swap(41, 43, array, mask, width)

	swap(44, 46, array, mask, width)
	swap(45, 47, array, mask, width)

	swap(48, 50, array, mask, width)
	swap(49, 51, array, mask, width)

	swap(52, 54, array, mask, width)
	swap(53, 55, array, mask, width)

	swap(56, 58, array, mask, width)
	swap(57, 59, array, mask, width)

	swap(60, 62, array, mask, width)
	swap(61, 63, array, mask, width)
	// 1x1 swap
	mask = 0x5555555555555555
	width = 1
	swap(0, 1, array, mask, width)

	swap(2, 3, array, mask, width)

	swap(4, 5, array, mask, width)

	swap(6, 7, array, mask, width)

	swap(8, 9, array, mask, width)

	swap(10, 11, array, mask, width)

	swap(12, 13, array, mask, width)

	swap(14, 15, array, mask, width)

	swap(16, 17, array, mask, width)

	swap(18, 19, array, mask, width)

	swap(20, 21, array, mask, width)

	swap(22, 23, array, mask, width)

	swap(24, 25, array, mask, width)

	swap(26, 27, array, mask, width)

	swap(28, 29, array, mask, width)

	swap(30, 31, array, mask, width)

	swap(32, 33, array, mask, width)

	swap(34, 35, array, mask, width)

	swap(36, 37, array, mask, width)

	swap(38, 39, array, mask, width)

	swap(40, 41, array, mask, width)

	swap(42, 43, array, mask, width)

	swap(44, 45, array, mask, width)

	swap(46, 47, array, mask, width)

	swap(48, 49, array, mask, width)

	swap(50, 51, array, mask, width)

	swap(52, 53, array, mask, width)

	swap(54, 55, array, mask, width)

	swap(56, 57, array, mask, width)

	swap(58, 59, array, mask, width)

	swap(60, 61, array, mask, width)

	swap(62, 63, array, mask, width)
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

func TransposeBitSets2(bmat []*bitset.BitSet) []*bitset.BitSet {
	m := uint(len(bmat))
	k := bmat[0].Len()

	// setup new matrix of BitSets to hold transposed values
	transposed := make([]*bitset.BitSet, k)
	for row := range transposed {
		transposed[row] = bitset.New(m)
	}

	// find set bits in original BitSets
	for i, b := range bmat {
		setBits := make([]uint, b.Count())
		b.NextSetMany(0, setBits)
		for _, s := range setBits {
			transposed[s].Set(uint(i))
		}
	}
	return transposed
}

/*
func ConcurrentSymmetricDifference(b *bitset.BitSet, compare *bitset.BitSet) (*bitset.BitSet, error) {
	if b.Len() != compare.Len() {
		return nil, fmt.Errorf("BitSets do not have the same length for XOR operations")
	}

	var wg sync.WaitGroup

	bcod := b.Bytes()
	bcom := compare.Bytes()
	n := len(bcod)
	result := make([]uint64, n)

	nThreads := 12
	if n < nThreads {
		nThreads = n
	}

	// number of blocks for which each goroutine is responsible
	nBlocks := n / nThreads

	// create ordered channels to store values from goroutines
	channels := make([]chan uint64, nBlocks)
	for i := 0; i < nThreads-1; i++ {
		channels[i] = make(chan uint64, nBlocks)
	}
	// last one may have excess columns
	channels[nThreads-1] = make(chan uint64, nBlocks+n%nThreads)

	// goroutine
	for i := 0; i < nThreads; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// handle extra blocks that didn't evenly divide
			var extraBlocks int
			if i == nThreads-1 {
				extraBlocks = n % nThreads
			}

			for c := 0; c < (nBlocks + extraBlocks); c++ {
				oBlock := bitset.From(bcod[c : c+1])
				oCompare := bitset.From(bcom[c : c+1])
				channels[i] <- oBlock.SymmetricDifference(oCompare).Bytes()[0]
			}
			close(channels[i])
		}(i)
	}

	// Wait until all goroutines have finished
	wg.Wait()

	// Reconstruct a transposed matrix from the channels
	for _, channel := range channels {
		var j int
		for block := range channel {
			result[j] = block
			j++
		}
	}

	return bitset.From(result), nil
}
*/

func ColumnarBitSetTranspose(matrix []*bitset.BitSet) []*bitset.BitSet {
	m := uint(len(matrix))
	n := matrix[0].Len()
	tr := make([]*bitset.BitSet, n)

	for col := uint(0); col < n; col++ {
		tr[col] = bitset.New(m)
		for row := uint(0); row < m; row++ {
			if matrix[row].Test(col) {
				tr[col].Set(row)
			}
		}
	}
	return tr
}

func ConcurrentColumnarBitSetTranspose(matrix []*bitset.BitSet) []*bitset.BitSet {
	var wg sync.WaitGroup
	m := len(matrix)
	n := int(matrix[0].Len())
	tr := make([]*bitset.BitSet, n)

	// the optimal number of goroutines will likely vary due to
	// hardware and array size
	// nThreads := n // one thread per column (likely only efficient for huge matrix)
	// nThreads := runtime.NumCPU()
	// nThreads := runtime.NumCPU()*2
	// 6 threads performs better in improved kkrt oprf but 12 performs better in benchmark
	nThreads := 6
	// add to quick check to ensure there are not more threads than columns
	if n < nThreads {
		nThreads = n
	}

	// number of columns for which each goroutine is responsible
	nColumns := n / nThreads

	// create ordered channels to store values from goroutines
	// each channel is buffered to store the number of desired rows
	channels := make([]chan *bitset.BitSet, nThreads)
	for i := 0; i < nThreads-1; i++ {
		channels[i] = make(chan *bitset.BitSet, nColumns)
	}
	// last one may have excess columns
	channels[nThreads-1] = make(chan *bitset.BitSet, nColumns+n%nThreads)

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
				row := bitset.New(uint(m))
				for r := 0; r < m; r++ {
					if matrix[r].Test(uint((i * nColumns) + c)) {
						row.Set(uint(r))
					}
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

// Attempt to make this cache-ideal but performance is actually worse and still has bugs
/*
func ConcurrentColumnarEncodedBitSetTranspose(matrix []*bitset.BitSet) []*bitset.BitSet {
	var wg sync.WaitGroup
	m := len(matrix)
	n := len(matrix[0].Bytes())
	//n := int(matrix[0].Len())
	tr := make([]*bitset.BitSet, n)

	// the optimal number of goroutines will likely vary due to
	// hardware and array size
	// nThreads := n // one thread per column (likely only efficient for huge matrix)
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
	channels := make([]chan *bitset.BitSet, nThreads)
	for i := 0; i < nThreads-1; i++ {
		channels[i] = make(chan *bitset.BitSet, nColumns)
	}
	// last one may have excess columns
	channels[nThreads-1] = make(chan *bitset.BitSet, nColumns+n%nThreads)

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
				row := bitset.New(uint(m)*64)
				for r := 0; r < m; r++ {
					row.Bytes()[r] = matrix[r].Bytes()[(i*nColumns)+c]
					//if matrix[r].Test(uint((i * nColumns) + c)) {
					//	row.Set(uint(r))
					//}
				}

				for y := 0; y < m; y++ {
					for z := 0; z < 64; z++ {

					}
					tmprow := bitset.New(uint(m))




					channels[i] <- tmprow
				}
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
*/
// ConcurrentColumnarBitSetSymmetricDifference operates on a BitSet matrix.
// An XOR is performed on each column manually against a generated BitSet.
// The result is stored in a new matrix and happes to be transposed.
//func ConcurrentColumnarBitSetSymmetricDifference(matrix []*bitset.BitSet, f func(*blake3.Hasher, *bitset.BitSet, int) *bitset.BitSet) []*bitset.BitSet {
func ConcurrentColumnarBitSetSymmetricDifference(matrix []*bitset.BitSet, f func(int) *bitset.BitSet) []*bitset.BitSet {
	var wg sync.WaitGroup
	m := len(matrix)
	n := int(matrix[0].Len())
	tr := make([]*bitset.BitSet, n)

	// the optimal number of goroutines will likely vary due to
	// hardware and array size
	// nThreads := n // one thread per column (likely only efficient for huge matrix)
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
	channels := make([]chan *bitset.BitSet, nThreads)
	for i := 0; i < nThreads-1; i++ {
		channels[i] = make(chan *bitset.BitSet, nColumns)
	}
	// last one may have excess columns
	channels[nThreads-1] = make(chan *bitset.BitSet, nColumns+n%nThreads)

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
				row := f(c).Clone()
				for r := 0; r < m; r++ {
					// Take XOR here
					if matrix[r].Test(uint((i*nColumns)+c)) != row.Test(uint(r)) {
						row.Set(uint(r))
					} else {
						row.Clear(uint(r))
					}
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

func ConcurrentColumnarBitSetSymmetricDifference2(matrix []*bitset.BitSet, f func(int) *bitset.BitSet) []*bitset.BitSet {
	var wg sync.WaitGroup
	m := len(matrix)
	n := int(matrix[0].Len())
	tr := make([]*bitset.BitSet, n)

	// the optimal number of goroutines will likely vary due to
	// hardware and array size
	// nThreads := n // one thread per column (likely only efficient for huge matrix)
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
	channels := make([]chan *bitset.BitSet, nThreads)
	for i := 0; i < nThreads-1; i++ {
		channels[i] = make(chan *bitset.BitSet, nColumns)
	}
	// last one may have excess columns
	channels[nThreads-1] = make(chan *bitset.BitSet, nColumns+n%nThreads)

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
				row := bitset.New(uint(m))
				for r := 0; r < m; r++ {
					if matrix[r].Test(uint((i * nColumns) + c)) {
						row.Set(uint(r))
					}
				}
				row.InPlaceSymmetricDifference(f(c))
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

// ConcurrentColumnarCacheSafeBitSetTranspose performs a transpose on a non-square BitSet matrix
// The cache-safe nature is due to limiting the number of rows which a goroutine can process sequentially.
// I am unsure whether this really will incur any benefit since ultimately you will have to rebuilt the row outside a goroutine.
// In fact, this may be less cache-efficient.
// With testing this method seems to be less efficient than the normal concurrent columnar transpose taking ~40% longer for 100 million by 64 BitSets
// TODO this method still has a bug for when you have less than 64 rows in the original BitSet matrix
func ConcurrentColumnarCacheSafeBitSetTranspose(matrix []*bitset.BitSet) []*bitset.BitSet {
	var wg sync.WaitGroup
	m := len(matrix)
	n := int(matrix[0].Len())
	tru := make([][]uint64, n) // to hold uint64 values encoding bits

	// the optimal number of goroutines will likely vary due to
	// hardware and array size
	// nThreads := n // one thread per column (likely only efficient for huge matrix)
	// nThreads := runtime.NumCPU()
	// nThreads := runtime.NumCPU()*2
	nThreads := 12
	// add to quick check to ensure there are not more threads than columns
	if n < nThreads {
		nThreads = n
	}

	// number of columns for which each goroutine is responsible
	nColumns := n / nThreads

	// number of uint64 to hold a given column of bits
	nUints := m / 64
	if m%64 > 0 {
		nUints += 1
	}

	// populate uint64 matrix
	for r := range tru {
		tru[r] = make([]uint64, nUints)
	}

	// create ordered channels to store values from goroutines
	// each channel is buffered to store the number of desired rows
	channels := make([]chan uint64, nThreads)
	for i := 0; i < nThreads-1; i++ {
		channels[i] = make(chan uint64, nColumns*nUints)
	}
	// last one may have excess columns
	channels[nThreads-1] = make(chan uint64, (nColumns+n%nThreads)*nUints)

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

			singleUint := bitset.New(64)
			for c := 0; c < (nColumns + extraColumns); c++ {
				for n := 0; n < nUints; n++ {
					singleUint.ClearAll()
					for r := 0; r < 64; r++ {
						if matrix[(64*n)+r].Test(uint((i * nColumns) + c)) {
							singleUint.Set(uint(r))
						}
					}
					channels[i] <- singleUint.Bytes()[0] // there will only ever be one uint64 to put
				}
			}

			close(channels[i])
		}(i)
	}

	// Wait until all goroutines have finished
	wg.Wait()

	// Reconstruct a transposed matrix from the channels
	for i, channel := range channels {
		var k int // row in channel
		var l int // index of integer in row
		for j := range channel {
			if l < nUints {
				tru[(i*nColumns)+k][l] = j
				l++
			} else {
				l = 0
				k++
				tru[(i*nColumns)+k][l] = j
				l++
			}
		}
	}

	// Convert transposed uint64 matrix back into bitset
	trb := UintsToBitSets(tru)

	return trb
}

// InPlaceSpliceBitSets takes two input BitSets and splices them together
// such that bo is spliced to the start of bt in place.
func InPlaceSpliceBitSets(bo, bt *bitset.BitSet) {
	indices := make([]uint, bo.Count())
	bo.NextSetMany(0, indices)
	for _, x := range indices {
		bt.Set(x + bo.Len())
	}
}

// SpliceBitSets2 takes two input BitSets and splices them together
// such that bo is spliced to the start of bt.
func SpliceBitSets2(bo, bt *bitset.BitSet) *bitset.BitSet {
	indices := make([]uint, bo.Count())
	bo.NextSetMany(0, indices)
	spliced := bt.Clone()

	for _, x := range indices {
		spliced.Set(x + bo.Len())
	}

	return spliced
}

// SpliceBitSets takes two input BitSets and splices them together
// such that bo is spliced to the start of bt.
// Example bo = 111, bt = 000 => 111000
// This method (appending uint64 slices) is faster than iterating
// through set bits (substantially so).
func SpliceBitSets(bo, bt *bitset.BitSet) *bitset.BitSet {
	uo := bo.Bytes()
	ut := bt.Bytes()

	uo = append(ut, uo...)

	return bitset.From(uo)
}

// ColumnarSymmetricDifference of a column from 2D base set and other set
// This is the BitSet equivalent of ^ (xor)
func ColumnarSymmetricDifference(b []*bitset.BitSet, compare *bitset.BitSet, column uint) (*bitset.BitSet, error) {
	if uint(len(b)) != compare.Len() {
		return nil, fmt.Errorf("2D BitSet column and compare BitSet do not have the same length for XOR operations")
	}

	result := bitset.New(uint(len(b)))

	for i, row := range b {
		var xor bool
		if row.Test(column) && compare.Test(uint(i)) {
			xor = false
		} else {
			xor = row.Test(column) || compare.Test(uint(i))
		}
		result.SetTo(uint(i), xor)
	}
	return result, nil
}

// InPlaceColumnarSymmetricDifference of a column from 2D base set and other set
// This is the BitSet equivalent of ^ (xor)
func InPlaceColumnarSymmetricDifference(b []*bitset.BitSet, compare *bitset.BitSet, column uint) error {
	if uint(len(b)) != compare.Len() {
		return fmt.Errorf("2D BitSet column and compare BitSet do not have the same length for XOR operations")
	}

	for i, row := range b {
		var xor bool
		if row.Test(column) && compare.Test(uint(i)) {
			xor = false
		} else {
			xor = row.Test(column) || compare.Test(uint(i))
		}
		row.SetTo(column, xor)
	}
	return nil
}

// expand BitSet matrix to make it square
func expandBitSets(bsets []*bitset.BitSet) []*bitset.BitSet {
	// due to nature of BitSet, this will be multiple of 64
	bLen := bsets[0].Len()
	expandBy := int(bLen) - len(bsets)

	if expandBy > 0 {
		expandBlock := make([]*bitset.BitSet, expandBy)
		for i := range expandBlock {
			expandBlock[i] = bitset.New(bLen)
		}

		bsets = append(bsets, expandBlock...)
	}

	return bsets
}

// expand BitSet matrix using the underlying uint64s
// this is faster than expanding directly with BitSets
func expandBitSetInts(bsets []*bitset.BitSet) [][]uint64 {
	numInts := int(bsets[0].Len()) / 64
	intSet := make([][]uint64, numInts*64)

	for i := range intSet {
		if i < len(bsets) {
			intSet[i] = bsets[i].Bytes()
		} else {
			intSet[i] = make([]uint64, numInts)
		}
	}

	return intSet
}

// expand BitSet matrix using the underlying uint64s
// this is faster than expanding directly with BitSets
// TODO each 64x64 bitset block is replaced by a row-ordered 64 uint64 slice
func expandBitSetToLinearInts(bsets []*bitset.BitSet) [][]uint64 {
	numInts := int(bsets[0].Len()) / 64
	intSet := make([][]uint64, numInts*64)

	for i := range intSet {
		if i < len(bsets) {
			intSet[i] = bsets[i].Bytes()
		} else {
			intSet[i] = make([]uint64, numInts)
		}
	}

	return intSet
}

// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
// This is MORE efficient that the other version
// This one iterates over the matrix and populates the
// contiguous row
func ContiguousBitSetTranspose(matrix []*bitset.BitSet) []*bitset.BitSet {
	m := len(matrix)
	k := int(matrix[0].Len())
	tr := make([]*bitset.BitSet, k)
	longRow := bitset.New(uint(m * k))

	for i := 0; i < m; i++ {
		for j := 0; j < k; j++ {
			if matrix[i].Test(uint(j)) {
				longRow.Set(uint(j*m + i))
			}
		}
	}

	longInts := longRow.Bytes()
	for i := range tr {
		tr[i] = bitset.From(longInts[i*(m/64) : (i+1)*(m/64)])
	}

	return tr
}

// no uint64 conversion
func ContiguousBitSetTranspose2(matrix []*bitset.BitSet) []*bitset.BitSet {
	m := len(matrix)
	k := int(matrix[0].Len())
	tr := make([]*bitset.BitSet, k)
	longRow := bitset.New(uint(m * k))

	for i := 0; i < m; i++ {
		for j := 0; j < k; j++ {
			if matrix[i].Test(uint(j)) {
				longRow.Set(uint(j*m + i))
			}
		}
	}

	for i := range tr {
		tr[i] = bitset.New(uint(m))
		for j := 0; j < m; j++ {
			if longRow.Test(uint(i*m + j)) {
				tr[i].Set(uint(j))
			}
		}
	}

	return tr
}

// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
// This is MORE efficient that the other version
// This one iterates over the matrix and populates the
// contiguous row
func ContiguousSparseBitSetTranspose(matrix []*bitset.BitSet) []*bitset.BitSet {
	m := len(matrix)
	k := int(matrix[0].Len())
	tr := make([]*bitset.BitSet, k)
	longRow := bitset.New(uint(m * k))

	for i := 0; i < m; i++ {
		setBits := make([]uint, matrix[i].Count())
		matrix[i].NextSetMany(0, setBits)
		for _, bit := range setBits {
			longRow.Set(bit*uint(m) + uint(i))
		}
	}

	longInts := longRow.Bytes()
	for i := range tr {
		tr[i] = bitset.From(longInts[i*(m/64) : (i+1)*(m/64)])
	}

	return tr
}

// Transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
// This is MORE efficient that the other version
// This one iterates over the matrix and populates the
// contiguous row
/*
func ContiguousSparseBitSetTranspose2(matrix []*bitset.BitSet) []*bitset.BitSet {
	m := len(matrix)
	k := int(matrix[0].Len())
	tr := make([]*bitset.BitSet, k)
	longRow := bitset.New(uint(m * k))

	for i := 0; i < m; i++ {
		bitsLeft := true
		index := uint(0)
		for bitsLeft == true {
			index, bitsLeft = matrix[i].NextSet(index)
			longRow.Set(index*uint(m) + uint(i))
		}
	}

	longInts := longRow.Bytes()
	for i := range tr {
		tr[i] = bitset.From(longInts[i*(m/64) : (i+1)*(m/64)])
	}

	return tr
}
*/
