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

// TODO this is not working currently!

// The major failing of my last attempt at concurrent transposition
// was that each goroutine (and there were far too many) was accessing
// the same shared array. This meant that the cache on each core had to
// be constantly updated as each coroutine updated their cache-local
// version. Instead this version splits everything by column. Each
// goroutine reads from the same shared matrix, but since nothing is
// is changed, the local cache shouldn't need an update. Then each
// goroutine sends the transposed row back to an ordered channel. Once
// all transpositions are done, rows are recombined into a 2D matrix.
func ConcurrentColumnarTranspose(matrix [][]uint8) [][]uint8 {
	var wg sync.WaitGroup
	m := len(matrix)
	n := len(matrix[0])
	tr := make([][]uint8, n)

	// the optimal number of goroutines will likely vary due to
	// hardware and array size
	nThreads := 1 // one thread per column (likely only efficient for huge matrix)
	// nThreads := runtime.NumCPU()
	// nThreads := runtime.NumCPU()*2

	wg.Add(nThreads)

	// number of columns for which each goroutine is responsible
	nColumns := n / nThreads
	var extraColumns int

	fmt.Println(nColumns)

	// create ordered channels to store values from goroutines
	// each channel is buffered to store the number of desired rows
	channels := make([]chan []uint8, nThreads)
	for i := 0; i < nThreads-1; i++ {
		channels[i] = make(chan []uint8, nColumns)
	}
	// last one may have excess columns
	channels[nThreads-1] = make(chan []uint8, nColumns+n%nThreads)

	// goroutine
	for i := 0; i < nThreads; i++ {
		go func(i int) {
			defer wg.Done()
			// we need to handle excess columns which don't evenly divide among
			// number of threads -> in this case, I just add to the last goroutine
			// perhaps a more sophisticated division of labor would be more efficient
			if i == nThreads {
				extraColumns = n % nThreads
			}

			row := make([]uint8, m)
			for c := 0; c < (nColumns + extraColumns); c++ {
				for r := 0; r < m; r++ {
					row[r] = matrix[r][(i*nColumns)+c]
					fmt.Println("go", row)
				}
				channels[i] <- row
			}
			close(channels[i])
		}(i)
	}

	// Wait until all goroutines have finished
	wg.Wait()

	// Reconstruct a transposed matrix from the channels
	/*
		for i, channel := range channels {
			fmt.Println("channel", i)

			var j int
			for row := range channel {
				tr[(i*nColumns)+j] = row
				fmt.Println("row", j, row)

				if i == nThreads-1 {
					if j == nColumns+extraColumns-1 {
						close(channel)
						fmt.Println("one")
					}
				} else {
					if j == nColumns-1 {
						close(channel)
						fmt.Println("two")
					}
				}

				j++
			}
	*/

	for i := 0; i < nThreads-1; i++ {
		fmt.Println("hit 1")
		for j := 0; j < nColumns-1; j++ {
			tr[(i*nColumns)+j] = <-channels[i]
		}
	}
	for j := 0; j < nColumns+extraColumns; j++ {
		fmt.Println("hit 2")
		tr[((nThreads-1)*nColumns)+j] = <-channels[nThreads-1]
	}

	for i := range matrix {
		fmt.Println(matrix[i])
	}
	fmt.Println("---tr---")
	for j := range tr {
		fmt.Println(tr[j])
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

// m x k to k x m
func transposeBitSets(bmat []*bitset.BitSet) []*bitset.BitSet {
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

func transposeBitSets2(bmat []*bitset.BitSet) []*bitset.BitSet {
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
