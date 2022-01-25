package util

import (
	"runtime"
	"sync"
)

const bitVectWidth = 512

// A BitVect is a matrix of 512 by 512 bits encoded into a contiguous slice of
// uint64 elements.
type BitVect struct {
	set [bitVectWidth * 8]uint64
}

// ConcurrentTransposeTall tranposes a tall (64 column) matrix. If the input
// matrix does not have a multiple of 512 rows (tall), panic. First it
// determines how many 512x512 bit blocks are necessary to contain the matrix.
// The blocks are divided among the number of workers. If there are fewer blocks
// than workers, this function operates as though it were single-threaded. For
// those small sets, performance could be improved by limiting the number of
// workers to the number of blocks but this incurs a performance penalty and it
// is much more likely that there will be more blocks than workers/cpu cores.
// Each goroutine, iterates over the blocks for which it is responsible. For
// each block it generates a BitVect from the matrix at the appropriate index,
// performs a cache-oblivious, in-place, contiguous transpose on the BitVect,
// and finally writes the result to a shared final output matrix. The last
// worker is responsible for any excess blocks which were not evenly divisible
// into the number of workers.
func ConcurrentTransposeTall(matrix [][]byte) [][]byte {
	if len(matrix)%bitVectWidth != 0 {
		panic("rows of input matrix not a multiple of 512")
	}

	nworkers := runtime.GOMAXPROCS(0)

	// number of blocks to split original matrix
	nblks := len(matrix) / bitVectWidth

	// how many blocks each worker is responsible for
	workerResp := nblks / nworkers

	// build output matrix
	trans := make([][]byte, bitVectWidth)
	for r := range trans {
		trans[r] = make([]byte, len(matrix)/8)
	}

	// Run a worker pool
	var wg sync.WaitGroup
	wg.Add(nworkers)
	for w := 0; w < nworkers; w++ {
		w := w
		go func() {
			defer wg.Done()
			step := workerResp * w
			var b BitVect
			if w == nworkers-1 { // last worker has extra work
				for i := step; i < nblks; i++ {
					b.unravelTall(matrix, i)
					b.transpose()
					b.ravelToWide(trans, i)
				}
			} else {
				for i := step; i < step+workerResp; i++ {
					b.unravelTall(matrix, i)
					b.transpose()
					b.ravelToWide(trans, i)
				}
			}
		}()
	}

	wg.Wait()

	return trans
}

// ConcurrentTransposeWide tranposes a wide (512 row) matrix. If the input
// matrix does not have a multiple of 64 columns (wide), panic. First it
// determines how many 512x512 bit blocks are necessary to contain the matrix.
// The blocks are divided among the number of workers. If there are fewer blocks
// than workers, this function operates as though it were single-threaded. For
// those small sets, performance could be improved by limiting the number of
// workers to the number of blocks but this incurs a performance penalty and it
// is much more likely that there will be more blocks than workers/cpu cores.
// Each goroutine iterates over the blocks for which it is responsible. For
// each block it generates a BitVect from the matrix at the appropriate index,
// performs a cache-oblivious, in-place, contiguous transpose on the BitVect,
// and finally writes the result to a shared final output matrix. The last
// worker is responsible for any excess blocks which were not evenly divisible
// into the number of workers.
func ConcurrentTransposeWide(matrix [][]byte) [][]byte {
	if len(matrix[0])%64 != 0 {
		panic("columns of input matrix not a multiple of 64")
	}

	nworkers := runtime.GOMAXPROCS(0)

	// determine number of blocks to split original matrix
	nblks := len(matrix[0]) / 64

	// how many blocks each worker is responsible for
	workerResp := nblks / nworkers

	// build output matrix
	trans := make([][]byte, len(matrix[0])*8)
	for r := range trans {
		trans[r] = make([]byte, 64)
	}

	// Run a worker pool
	var wg sync.WaitGroup
	wg.Add(nworkers)
	for w := 0; w < nworkers; w++ {
		w := w
		go func() {
			defer wg.Done()
			step := workerResp * w
			var b BitVect
			if w == nworkers-1 { // last worker has extra work
				for i := step; i < nblks; i++ {
					b.unravelWide(matrix, i)
					b.transpose()
					b.ravelToTall(trans, i)
				}
			} else {
				for i := step; i < step+workerResp; i++ {
					b.unravelWide(matrix, i)
					b.transpose()
					b.ravelToTall(trans, i)
				}
			}
		}()
	}

	wg.Wait()

	return trans
}

// transpose performs a cache-oblivious, in-place, contiguous transpose.
// Since a BitVect represents a 512 by 512 square bit matrix, transposition will
// be performed blockwise starting with blocks of 256 x 4, swapped about the
// principle diagonal. Then block size will decrease by half until it is only
// 64 x 1. The remaining transposition steps are performed using bit masks and
// shifts. Operations are performed on blocks of bits of size 32, 16, 8, 4, 2,
// and 1. Since the input is square, the transposition can be performed in place.
func (b *BitVect) transpose() {
	tmp := make([]uint64, 4)
	// Transpose 4 x 256 blocks
	var jmp int
	for i := 0; i < 256; i++ {
		jmp = i * 8
		copy(tmp, b.set[jmp+4:jmp+8])
		copy(b.set[jmp+4:jmp+8], b.set[(256*8)+jmp:(256*8)+jmp+4])
		copy(b.set[(256*8)+jmp:(256*8)+jmp+4], tmp)
	}

	// Transpose 2 x 128 blocks
	for j := 0; j < 128; j++ {
		jmp = j * 8
		copy(tmp, b.set[jmp+2:jmp+4])
		copy(b.set[jmp+2:jmp+4], b.set[(128*8)+jmp:(128*8)+jmp+2])
		copy(b.set[(128*8)+jmp:(128*8)+jmp+2], tmp[:2])

		copy(tmp, b.set[jmp+6:jmp+8])
		copy(b.set[jmp+6:jmp+8], b.set[(128*8)+jmp+4:(128*8)+jmp+6])
		copy(b.set[(128*8)+jmp+4:(128*8)+jmp+6], tmp[:2])

		copy(tmp, b.set[(256*8)+jmp+2:(256*8)+jmp+4])
		copy(b.set[(256*8)+jmp+2:(256*8)+jmp+4], b.set[(384*8)+jmp:(384*8)+jmp+2])
		copy(b.set[(384*8)+jmp:(384*8)+jmp+2], tmp[:2])

		copy(tmp, b.set[(256*8)+jmp+6:(256*8)+jmp+8])
		copy(b.set[(256*8)+jmp+6:(256*8)+jmp+8], b.set[(384*8)+jmp+4:(384*8)+jmp+6])
		copy(b.set[(384*8)+jmp+4:(384*8)+jmp+6], tmp[:2])
	}

	// Transpose 1 x 64 blocks
	for k := 0; k < 64; k++ {
		jmp = k * 8
		copy(tmp, b.set[jmp+1:jmp+2])
		copy(b.set[jmp+1:jmp+2], b.set[(64*8)+jmp:(64*8)+jmp+1])
		copy(b.set[(64*8)+jmp:(64*8)+jmp+1], tmp[:1])

		copy(tmp, b.set[jmp+3:jmp+4])
		copy(b.set[jmp+3:jmp+4], b.set[(64*8)+jmp+2:(64*8)+jmp+3])
		copy(b.set[(64*8)+jmp+2:(64*8)+jmp+3], tmp[:1])

		copy(tmp, b.set[jmp+5:jmp+6])
		copy(b.set[jmp+5:jmp+6], b.set[(64*8)+jmp+4:(64*8)+jmp+5])
		copy(b.set[(64*8)+jmp+4:(64*8)+jmp+5], tmp[:1])

		copy(tmp, b.set[jmp+7:jmp+8])
		copy(b.set[jmp+7:jmp+8], b.set[(64*8)+jmp+6:(64*8)+jmp+7])
		copy(b.set[(64*8)+jmp+6:(64*8)+jmp+7], tmp[:1])

		copy(tmp, b.set[(128*8)+jmp+1:(128*8)+jmp+2])
		copy(b.set[(128*8)+jmp+1:(128*8)+jmp+2], b.set[(192*8)+jmp:(192*8)+jmp+1])
		copy(b.set[(192*8)+jmp:(192*8)+jmp+1], tmp[:1])

		copy(tmp, b.set[(128*8)+jmp+3:(128*8)+jmp+4])
		copy(b.set[(128*8)+jmp+3:(128*8)+jmp+4], b.set[(192*8)+jmp+2:(192*8)+jmp+3])
		copy(b.set[(192*8)+jmp+2:(192*8)+jmp+3], tmp[:1])

		copy(tmp, b.set[(128*8)+jmp+5:(128*8)+jmp+6])
		copy(b.set[(128*8)+jmp+5:(128*8)+jmp+6], b.set[(192*8)+jmp+4:(192*8)+jmp+5])
		copy(b.set[(192*8)+jmp+4:(192*8)+jmp+5], tmp[:1])

		copy(tmp, b.set[(128*8)+jmp+7:(128*8)+jmp+8])
		copy(b.set[(128*8)+jmp+7:(128*8)+jmp+8], b.set[(192*8)+jmp+6:(192*8)+jmp+7])
		copy(b.set[(192*8)+jmp+6:(192*8)+jmp+7], tmp[:1])

		copy(tmp, b.set[(256*8)+jmp+1:(256*8)+jmp+2])
		copy(b.set[(256*8)+jmp+1:(256*8)+jmp+2], b.set[(320*8)+jmp:(320*8)+jmp+1])
		copy(b.set[(320*8)+jmp:(320*8)+jmp+1], tmp[:1])

		copy(tmp, b.set[(256*8)+jmp+3:(256*8)+jmp+4])
		copy(b.set[(256*8)+jmp+3:(256*8)+jmp+4], b.set[(320*8)+jmp+2:(320*8)+jmp+3])
		copy(b.set[(320*8)+jmp+2:(320*8)+jmp+3], tmp[:1])

		copy(tmp, b.set[(256*8)+jmp+5:(256*8)+jmp+6])
		copy(b.set[(256*8)+jmp+5:(256*8)+jmp+6], b.set[(320*8)+jmp+4:(320*8)+jmp+5])
		copy(b.set[(320*8)+jmp+4:(320*8)+jmp+5], tmp[:1])

		copy(tmp, b.set[(256*8)+jmp+7:(256*8)+jmp+8])
		copy(b.set[(256*8)+jmp+7:(256*8)+jmp+8], b.set[(320*8)+jmp+6:(320*8)+jmp+7])
		copy(b.set[(320*8)+jmp+6:(320*8)+jmp+7], tmp[:1])

		copy(tmp, b.set[(384*8)+jmp+1:(384*8)+jmp+2])
		copy(b.set[(384*8)+jmp+1:(384*8)+jmp+2], b.set[(448*8)+jmp:(448*8)+jmp+1])
		copy(b.set[(448*8)+jmp:(448*8)+jmp+1], tmp[:1])

		copy(tmp, b.set[(384*8)+jmp+3:(384*8)+jmp+4])
		copy(b.set[(384*8)+jmp+3:(384*8)+jmp+4], b.set[(448*8)+jmp+2:(448*8)+jmp+3])
		copy(b.set[(448*8)+jmp+2:(448*8)+jmp+3], tmp[:1])

		copy(tmp, b.set[(384*8)+jmp+5:(384*8)+jmp+6])
		copy(b.set[(384*8)+jmp+5:(384*8)+jmp+6], b.set[(448*8)+jmp+4:(448*8)+jmp+5])
		copy(b.set[(448*8)+jmp+4:(448*8)+jmp+5], tmp[:1])

		copy(tmp, b.set[(384*8)+jmp+7:(384*8)+jmp+8])
		copy(b.set[(384*8)+jmp+7:(384*8)+jmp+8], b.set[(448*8)+jmp+6:(448*8)+jmp+7])
		copy(b.set[(448*8)+jmp+6:(448*8)+jmp+7], tmp[:1])

	}

	// Bitwise transposition
	for blk := 0; blk < 8; blk++ {
		for col := 0; col < 8; col++ {
			transpose64(b, blk, col)
		}
	}
}

// swap swaps two rows of masked binary elements in a 64x64 bit matrix which is
// held as a contiguous uint64 array in a BitVect.
func swap(a, b int, vect *BitVect, mask uint64, width int) {
	t := (vect.set[a] ^ (vect.set[b] << width)) & mask
	vect.set[a] = vect.set[a] ^ t
	vect.set[b] = vect.set[b] ^ (t >> width)
}

// transpose64 performs a bitwise transpose on a 64x64 bit matrix which
// is held as a contiguous uint64 array in a BitVect. Instead of looping and
// generating the mask with each loop, the unrolled version is fully declared
// which boosts performance at the expense of verbosity.
func transpose64(vect *BitVect, vblock, col int) {
	jmp := vblock*(64*8) + col
	// 32x32 swap
	var mask uint64 = 0xFFFFFFFF00000000
	var width int = 32
	swap(jmp+(8*0), jmp+(8*32), vect, mask, width)  // 0 and 32
	swap(jmp+(8*1), jmp+(8*33), vect, mask, width)  // 1 and 33
	swap(jmp+(8*2), jmp+(8*34), vect, mask, width)  // 2 and 34
	swap(jmp+(8*3), jmp+(8*35), vect, mask, width)  // 3 and 35
	swap(jmp+(8*4), jmp+(8*36), vect, mask, width)  // 4 and 36
	swap(jmp+(8*5), jmp+(8*37), vect, mask, width)  // 5 and 37
	swap(jmp+(8*6), jmp+(8*38), vect, mask, width)  // 6 and 38
	swap(jmp+(8*7), jmp+(8*39), vect, mask, width)  // 7 and 39
	swap(jmp+(8*8), jmp+(8*40), vect, mask, width)  // 8 and 40
	swap(jmp+(8*9), jmp+(8*41), vect, mask, width)  // 9 and 41
	swap(jmp+(8*10), jmp+(8*42), vect, mask, width) // 10 and 42
	swap(jmp+(8*11), jmp+(8*43), vect, mask, width) // 11 and 43
	swap(jmp+(8*12), jmp+(8*44), vect, mask, width) // 12 and 44
	swap(jmp+(8*13), jmp+(8*45), vect, mask, width) // 13 and 45
	swap(jmp+(8*14), jmp+(8*46), vect, mask, width) // 14 and 46
	swap(jmp+(8*15), jmp+(8*47), vect, mask, width) // 15 and 47
	swap(jmp+(8*16), jmp+(8*48), vect, mask, width) // 16 and 48
	swap(jmp+(8*17), jmp+(8*49), vect, mask, width) // 17 and 49
	swap(jmp+(8*18), jmp+(8*50), vect, mask, width) // 18 and 50
	swap(jmp+(8*19), jmp+(8*51), vect, mask, width) // 19 and 51
	swap(jmp+(8*20), jmp+(8*52), vect, mask, width) // 20 and 52
	swap(jmp+(8*21), jmp+(8*53), vect, mask, width) // 21 and 53
	swap(jmp+(8*22), jmp+(8*54), vect, mask, width) // 22 and 54
	swap(jmp+(8*23), jmp+(8*55), vect, mask, width) // 23 and 55
	swap(jmp+(8*24), jmp+(8*56), vect, mask, width) // 24 and 56
	swap(jmp+(8*25), jmp+(8*57), vect, mask, width) // 25 and 57
	swap(jmp+(8*26), jmp+(8*58), vect, mask, width) // 26 and 58
	swap(jmp+(8*27), jmp+(8*59), vect, mask, width) // 27 and 29
	swap(jmp+(8*28), jmp+(8*60), vect, mask, width) // 28 and 60
	swap(jmp+(8*29), jmp+(8*61), vect, mask, width) // 29 and 61
	swap(jmp+(8*30), jmp+(8*62), vect, mask, width) // 30 and 62
	swap(jmp+(8*31), jmp+(8*63), vect, mask, width) // 31 and 63
	// 16x16 swap
	mask = 0xFFFF0000FFFF0000
	width = 16
	swap(jmp+(8*0), jmp+(8*16), vect, mask, width)  // 0 and 16
	swap(jmp+(8*1), jmp+(8*17), vect, mask, width)  // 1 and 17
	swap(jmp+(8*2), jmp+(8*18), vect, mask, width)  // 2 and 18
	swap(jmp+(8*3), jmp+(8*19), vect, mask, width)  // 3 and 19
	swap(jmp+(8*4), jmp+(8*20), vect, mask, width)  // 4 and 20
	swap(jmp+(8*5), jmp+(8*21), vect, mask, width)  // 5 and 21
	swap(jmp+(8*6), jmp+(8*22), vect, mask, width)  // 6 and 22
	swap(jmp+(8*7), jmp+(8*23), vect, mask, width)  // 7 and 23
	swap(jmp+(8*8), jmp+(8*24), vect, mask, width)  // 8 and 24
	swap(jmp+(8*9), jmp+(8*25), vect, mask, width)  // 9 and 25
	swap(jmp+(8*10), jmp+(8*26), vect, mask, width) // 10 and 26
	swap(jmp+(8*11), jmp+(8*27), vect, mask, width) // 11 and 27
	swap(jmp+(8*12), jmp+(8*28), vect, mask, width) // 12 and 28
	swap(jmp+(8*13), jmp+(8*29), vect, mask, width) // 13 and 29
	swap(jmp+(8*14), jmp+(8*30), vect, mask, width) // 14 and 30
	swap(jmp+(8*15), jmp+(8*31), vect, mask, width) // 15 and 31

	swap(jmp+(8*32), jmp+(8*48), vect, mask, width) // 32 and 48
	swap(jmp+(8*33), jmp+(8*49), vect, mask, width) // 33 and 49
	swap(jmp+(8*34), jmp+(8*50), vect, mask, width) // 34 and 50
	swap(jmp+(8*35), jmp+(8*51), vect, mask, width) // 35 and 51
	swap(jmp+(8*36), jmp+(8*52), vect, mask, width) // 36 and 52
	swap(jmp+(8*37), jmp+(8*53), vect, mask, width) // 37 and 53
	swap(jmp+(8*38), jmp+(8*54), vect, mask, width) // 38 and 54
	swap(jmp+(8*39), jmp+(8*55), vect, mask, width) // 39 and 55
	swap(jmp+(8*40), jmp+(8*56), vect, mask, width) // 40 and 56
	swap(jmp+(8*41), jmp+(8*57), vect, mask, width) // 41 and 57
	swap(jmp+(8*42), jmp+(8*58), vect, mask, width) // 42 and 58
	swap(jmp+(8*43), jmp+(8*59), vect, mask, width) // 43 and 59
	swap(jmp+(8*44), jmp+(8*60), vect, mask, width) // 44 and 60
	swap(jmp+(8*45), jmp+(8*61), vect, mask, width) // 45 and 61
	swap(jmp+(8*46), jmp+(8*62), vect, mask, width) // 46 and 62
	swap(jmp+(8*47), jmp+(8*63), vect, mask, width) // 47 and 63
	// 8x8 swap
	mask = 0xFF00FF00FF00FF00
	width = 8
	swap(jmp+(8*0), jmp+(8*8), vect, mask, width)  // 0 and 8
	swap(jmp+(8*1), jmp+(8*9), vect, mask, width)  // 1 and 9
	swap(jmp+(8*2), jmp+(8*10), vect, mask, width) // 2 and 10
	swap(jmp+(8*3), jmp+(8*11), vect, mask, width) // 3 and 11
	swap(jmp+(8*4), jmp+(8*12), vect, mask, width) // 4 and 12
	swap(jmp+(8*5), jmp+(8*13), vect, mask, width) // 5 and 13
	swap(jmp+(8*6), jmp+(8*14), vect, mask, width) // 6 and 14
	swap(jmp+(8*7), jmp+(8*15), vect, mask, width) // 7 and 15

	swap(jmp+(8*16), jmp+(8*24), vect, mask, width) // 16 and 24
	swap(jmp+(8*17), jmp+(8*25), vect, mask, width) // 17 and 25
	swap(jmp+(8*18), jmp+(8*26), vect, mask, width) // 18 and 26
	swap(jmp+(8*19), jmp+(8*27), vect, mask, width) // 19 and 27
	swap(jmp+(8*20), jmp+(8*28), vect, mask, width) // 20 and 28
	swap(jmp+(8*21), jmp+(8*29), vect, mask, width) // 21 and 29
	swap(jmp+(8*22), jmp+(8*30), vect, mask, width) // 22 and 30
	swap(jmp+(8*23), jmp+(8*31), vect, mask, width) // 23 and 31

	swap(jmp+(8*32), jmp+(8*40), vect, mask, width) // 32 and 40
	swap(jmp+(8*33), jmp+(8*41), vect, mask, width) // 33 and 41
	swap(jmp+(8*34), jmp+(8*42), vect, mask, width) // 34 and 42
	swap(jmp+(8*35), jmp+(8*43), vect, mask, width) // 35 and 43
	swap(jmp+(8*36), jmp+(8*44), vect, mask, width) // 36 and 44
	swap(jmp+(8*37), jmp+(8*45), vect, mask, width) // 37 and 45
	swap(jmp+(8*38), jmp+(8*46), vect, mask, width) // 38 and 46
	swap(jmp+(8*39), jmp+(8*47), vect, mask, width) // 39 and 47

	swap(jmp+(8*48), jmp+(8*56), vect, mask, width) // 48 and 56
	swap(jmp+(8*49), jmp+(8*57), vect, mask, width) // 49 and 57
	swap(jmp+(8*50), jmp+(8*58), vect, mask, width) // 50 and 58
	swap(jmp+(8*51), jmp+(8*59), vect, mask, width) // 51 and 59
	swap(jmp+(8*52), jmp+(8*60), vect, mask, width) // 52 and 60
	swap(jmp+(8*53), jmp+(8*61), vect, mask, width) // 53 and 61
	swap(jmp+(8*54), jmp+(8*62), vect, mask, width) // 54 and 62
	swap(jmp+(8*55), jmp+(8*63), vect, mask, width) // 55 and 63
	// 4x4 swap
	mask = 0xF0F0F0F0F0F0F0F0
	width = 4
	swap(jmp+(8*0), jmp+(8*4), vect, mask, width) // 0 and 4
	swap(jmp+(8*1), jmp+(8*5), vect, mask, width) // 1 and 5
	swap(jmp+(8*2), jmp+(8*6), vect, mask, width) // 2 and 6
	swap(jmp+(8*3), jmp+(8*7), vect, mask, width) // 3 and 7

	swap(jmp+(8*8), jmp+(8*12), vect, mask, width)  // 8 and 12
	swap(jmp+(8*9), jmp+(8*13), vect, mask, width)  // 9 and 13
	swap(jmp+(8*10), jmp+(8*14), vect, mask, width) // 10 and 14
	swap(jmp+(8*11), jmp+(8*15), vect, mask, width) // 11 and 15

	swap(jmp+(8*16), jmp+(8*20), vect, mask, width) // 16 and 20
	swap(jmp+(8*17), jmp+(8*21), vect, mask, width) // 17 and 21
	swap(jmp+(8*18), jmp+(8*22), vect, mask, width) // 18 and 22
	swap(jmp+(8*19), jmp+(8*23), vect, mask, width) // 19 and 23

	swap(jmp+(8*24), jmp+(8*28), vect, mask, width) // 24 and 28
	swap(jmp+(8*25), jmp+(8*29), vect, mask, width) // 25 and 29
	swap(jmp+(8*26), jmp+(8*30), vect, mask, width) // 26 and 30
	swap(jmp+(8*27), jmp+(8*31), vect, mask, width) // 27 and 31

	swap(jmp+(8*32), jmp+(8*36), vect, mask, width) // 32 and 36
	swap(jmp+(8*33), jmp+(8*37), vect, mask, width) // 33 and 37
	swap(jmp+(8*34), jmp+(8*38), vect, mask, width) // 34 and 38
	swap(jmp+(8*35), jmp+(8*39), vect, mask, width) // 35 and 39

	swap(jmp+(8*40), jmp+(8*44), vect, mask, width) // 40 and 44
	swap(jmp+(8*41), jmp+(8*45), vect, mask, width) // 41 and 45
	swap(jmp+(8*42), jmp+(8*46), vect, mask, width) // 42 and 46
	swap(jmp+(8*43), jmp+(8*47), vect, mask, width) // 43 and 47

	swap(jmp+(8*48), jmp+(8*52), vect, mask, width) // 48 and 52
	swap(jmp+(8*49), jmp+(8*53), vect, mask, width) // 49 and 53
	swap(jmp+(8*50), jmp+(8*54), vect, mask, width) // 50 and 54
	swap(jmp+(8*51), jmp+(8*55), vect, mask, width) // 51 and 55

	swap(jmp+(8*56), jmp+(8*60), vect, mask, width) // 56 and 60
	swap(jmp+(8*57), jmp+(8*61), vect, mask, width) // 57 and 61
	swap(jmp+(8*58), jmp+(8*62), vect, mask, width) // 58 and 62
	swap(jmp+(8*59), jmp+(8*63), vect, mask, width) // 59 and 63
	// 2x2 swap
	mask = 0xcccccccccccccccc
	width = 2
	swap(jmp+(8*0), jmp+(8*2), vect, mask, width) // 0 and 2
	swap(jmp+(8*1), jmp+(8*3), vect, mask, width) // 1 and 3

	swap(jmp+(8*4), jmp+(8*6), vect, mask, width) // 4 and 6
	swap(jmp+(8*5), jmp+(8*7), vect, mask, width) // 5 and 7

	swap(jmp+(8*8), jmp+(8*10), vect, mask, width) // 8 and 10
	swap(jmp+(8*9), jmp+(8*11), vect, mask, width) // 9 and 11

	swap(jmp+(8*12), jmp+(8*14), vect, mask, width) // 12 and 14
	swap(jmp+(8*13), jmp+(8*15), vect, mask, width) // 13 and 15

	swap(jmp+(8*16), jmp+(8*18), vect, mask, width) // 16 and 18
	swap(jmp+(8*17), jmp+(8*19), vect, mask, width) // 17 and 19

	swap(jmp+(8*20), jmp+(8*22), vect, mask, width) // 20 and 22
	swap(jmp+(8*21), jmp+(8*23), vect, mask, width) // 21 and 23

	swap(jmp+(8*24), jmp+(8*26), vect, mask, width) // 24 and 26
	swap(jmp+(8*25), jmp+(8*27), vect, mask, width) // 25 and 27

	swap(jmp+(8*28), jmp+(8*30), vect, mask, width) // 28 and 30
	swap(jmp+(8*29), jmp+(8*31), vect, mask, width) // 29 and 31

	swap(jmp+(8*32), jmp+(8*34), vect, mask, width) // 32 and 34
	swap(jmp+(8*33), jmp+(8*35), vect, mask, width) // 33 and 35

	swap(jmp+(8*36), jmp+(8*38), vect, mask, width) // 36 and 38
	swap(jmp+(8*37), jmp+(8*39), vect, mask, width) // 37 and 39

	swap(jmp+(8*40), jmp+(8*42), vect, mask, width) // 40 and 42
	swap(jmp+(8*41), jmp+(8*43), vect, mask, width) // 41 and 43

	swap(jmp+(8*44), jmp+(8*46), vect, mask, width) // 44 and 46
	swap(jmp+(8*45), jmp+(8*47), vect, mask, width) // 45 and 47

	swap(jmp+(8*48), jmp+(8*50), vect, mask, width) // 48 and 50
	swap(jmp+(8*49), jmp+(8*51), vect, mask, width) // 49 and 51

	swap(jmp+(8*52), jmp+(8*54), vect, mask, width) // 52 and 54
	swap(jmp+(8*53), jmp+(8*55), vect, mask, width) // 53 and 55

	swap(jmp+(8*56), jmp+(8*58), vect, mask, width) // 56 and 58
	swap(jmp+(8*57), jmp+(8*59), vect, mask, width) // 57 and 59

	swap(jmp+(8*60), jmp+(8*62), vect, mask, width) // 60 and 62
	swap(jmp+(8*61), jmp+(8*63), vect, mask, width) // 61 and 63
	// 1x1 swap
	mask = 0xaaaaaaaaaaaaaaaa
	width = 1
	swap(jmp+(8*0), jmp+(8*1), vect, mask, width) // 0 and 1

	swap(jmp+(8*2), jmp+(8*3), vect, mask, width) // 2 and 3

	swap(jmp+(8*4), jmp+(8*5), vect, mask, width) // 4 and 5

	swap(jmp+(8*6), jmp+(8*7), vect, mask, width) // 6 and 7

	swap(jmp+(8*8), jmp+(8*9), vect, mask, width) // 8 and 9

	swap(jmp+(8*10), jmp+(8*11), vect, mask, width) // 10 and 11

	swap(jmp+(8*12), jmp+(8*13), vect, mask, width) // 12 and 13

	swap(jmp+(8*14), jmp+(8*15), vect, mask, width) // 14 and 15

	swap(jmp+(8*16), jmp+(8*17), vect, mask, width) // 16 and 17

	swap(jmp+(8*18), jmp+(8*19), vect, mask, width) // 18 and 19

	swap(jmp+(8*20), jmp+(8*21), vect, mask, width) // 20 and 21

	swap(jmp+(8*22), jmp+(8*23), vect, mask, width) // 22 and 23

	swap(jmp+(8*24), jmp+(8*25), vect, mask, width) // 24 and 25

	swap(jmp+(8*26), jmp+(8*27), vect, mask, width) // 26 and 27

	swap(jmp+(8*28), jmp+(8*29), vect, mask, width) // 28 and 29

	swap(jmp+(8*30), jmp+(8*31), vect, mask, width) // 30 and 31

	swap(jmp+(8*32), jmp+(8*33), vect, mask, width) // 32 and 33

	swap(jmp+(8*34), jmp+(8*35), vect, mask, width) // 34 and 35

	swap(jmp+(8*36), jmp+(8*37), vect, mask, width) // 36 and 37

	swap(jmp+(8*38), jmp+(8*39), vect, mask, width) // 38 and 39

	swap(jmp+(8*40), jmp+(8*41), vect, mask, width) // 40 and 41

	swap(jmp+(8*42), jmp+(8*43), vect, mask, width) // 42 and 43

	swap(jmp+(8*44), jmp+(8*45), vect, mask, width) // 44 and 45

	swap(jmp+(8*46), jmp+(8*47), vect, mask, width) // 46 and 47

	swap(jmp+(8*48), jmp+(8*49), vect, mask, width) // 48 and 49

	swap(jmp+(8*50), jmp+(8*51), vect, mask, width) // 50 and 51

	swap(jmp+(8*52), jmp+(8*53), vect, mask, width) // 52 and 53

	swap(jmp+(8*54), jmp+(8*55), vect, mask, width) // 54 and 55

	swap(jmp+(8*56), jmp+(8*57), vect, mask, width) // 56 and 57

	swap(jmp+(8*58), jmp+(8*59), vect, mask, width) // 58 and 59

	swap(jmp+(8*60), jmp+(8*61), vect, mask, width) // 60 and 61

	swap(jmp+(8*62), jmp+(8*63), vect, mask, width) // 62 and 63
}
