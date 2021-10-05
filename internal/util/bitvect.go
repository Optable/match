package util

import (
	"fmt"
	"sync"
)

// A BitVect is a matrix of 512 by 512 bits encoded into uint64 elements.
type BitVect struct {
	set [512 * 8]uint64
}

// unravelTall is a constructor used to create a BitVect from a 2D matrix of uint64.
// The matrix must have 8 columns and 512 rows. idx is the block target.
func unravelTall(matrix [][]uint64, idx int) BitVect {
	set := [4096]uint64{}

	for i := 0; i < 512; i++ {
		copy(set[(i)*8:(i+1)*8], matrix[(512*idx)+i])
	}
	return BitVect{set}
}

// unravelWide is a constructor used to create a BitVect from a 2D matrix of uint64.
// The matrix must have 8 columns and 512 rows. idx is the block target.
func unravelWide(matrix [][]uint64, idx int) BitVect {
	set := [4096]uint64{}

	for i := 0; i < 512; i++ {
		copy(set[i*8:(i+1)*8], matrix[i][idx*8:(8*idx)+8])
	}
	return BitVect{set}
}

// ravelToTall reconstructs a subsection of a tall (mx8) matrix from a BitVect.
func (b BitVect) ravelToTall(matrix [][]uint64, idx int) {
	for i := 0; i < 512; i++ {
		copy(matrix[(idx*512)+i][:], b.set[i*8:(i+1)*8])
	}
}

// ravelToWide reconstructs a subsection of a wide (512xn) matrix from a BitVect.
func (b BitVect) ravelToWide(matrix [][]uint64, idx int) {
	for i := 0; i < 512; i++ {
		copy(matrix[i][idx*8:(idx+1)*8], b.set[(i*8):(i+1)*8])
	}
}

// printBits prints a subset of the overall bit array. The limit is in bits but
// since we are really printing uint64, everything is rounded down to the nearest
// multiple of 64. For example: b.PrintBits(66) will print a 64x64 bit array.
func (b BitVect) printBits(lim int) {
	//lim = lim/64
	if lim > 512 {
		lim = 512
	}

	for i := 0; i < lim; i++ {
		fmt.Printf("%064b\n", b.set[i*8:(i*8)+(lim/64)])
	}
}

// printUints prints all of the 512x8 uints in the bit array. Good for testing
// transpose operations performed prior to the bit level.
func (b BitVect) printUints() {
	for i := 0; i < 512; i++ {
		fmt.Println(i, " - ", b.set[i*8:(i+1)*8])
	}
}

// checkBit checks if a single bit in a uint64 is set
func checkBit(u uint64, i uint) bool {
	return u&(1<<i) > 0 // AND with mask with single set bit at testing location
}

/* TODO - manual check not working, using compare with doubly-transposed to confirm
// CheckTranspose compares BitVect to second BitVect to determined if they are
// the transposed matrix of each other.
func (b BitVect) CheckTranspose(t BitVect) bool {
	fmt.Println("test")
	for r := uint(0); r < 512; r++ {
		for c := uint(0); c < 8; c++ {
			for i := uint(0); i < 64; i++ {
				if !checkBit(b.set[(r*8)+c], i) && !checkBit(t.set[(((64*c)+(64-i))*8)+(r/64)], r%64) {
					fmt.Println("row", r, "col", c, "bit", i, "not in proper transposed position!")
					return false
				}
				fmt.Println("row", r, "col", c, "bit", i, "transposed properly")
			}
		}
	}

	return true
}
*/

// ConcurrentTranspose tranposes a wide (512 row) or tall (8 column) matrix.
// First it determines how many 512x512 bit blocks are necessary to contain the
// matrix and hold the indices where the blocks should be split from the larger
// matrix. The input matrix must have a multiple of 512 rows (tall) or 8 columns (wide)
// The indices are passed into a channel which is being read by a worker pool of
// goroutines. Each goroutine reads an index, generates a BitVect from the matrix
// at that index (with padding if necessary), performs a cache-oblivious, in-place,
// contiguous transpose on the BitVect, and finally writes the result to a shared
// final output matrix.
func ConcurrentTranspose(matrix [][]uint64, nworkers int) [][]uint64 {
	// determine is original matrix is wide or tall
	var tall bool
	if len(matrix[0]) == 8 {
		tall = true
	}

	// determine number of blocks to split original matrix
	var nblks int
	if tall {
		nblks = len(matrix) / 512
	} else {
		nblks = len(matrix[0]) / 8
	}

	// build output matrix
	var nrows, ncols int
	if tall {
		nrows = 512
		ncols = len(matrix) / 64
	} else {
		nrows = len(matrix[0]) * 64
		ncols = 8
	}
	trans := make([][]uint64, nrows)
	for r := range trans {
		trans[r] = make([]uint64, ncols)
	}

	// feed into buffered channel
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
			if tall {
				for id := range ch {
					b := unravelTall(matrix, id)
					b.transpose()
					b.ravelToWide(trans, id)
				}
			} else {
				for id := range ch {
					b := unravelWide(matrix, id)
					b.transpose()
					b.ravelToTall(trans, id)
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
// principle diagonal. Then blocks size will decrease by half until it is only
// 64 x 1. The remaining transposition steps are performed using bit masks and
// shifts. Operations are performed on blocks of bits of size 32, 16, 8, 4, 2,
// and 1. Since the input is square, the transposition can be performed in place.
func (b *BitVect) transpose() {
	// Transpose 4 x 256 blocks

	tmp4 := make([]uint64, 4)
	var jmp int
	for i := 0; i < 256; i++ {
		jmp = i * 8
		copy(tmp4, b.set[jmp+4:jmp+8])
		copy(b.set[jmp+4:jmp+8], b.set[(256*8)+jmp:(256*8)+jmp+4])
		copy(b.set[(256*8)+jmp:(256*8)+jmp+4], tmp4)
	}

	// Transpose 2 x 128 blocks
	tmp2 := make([]uint64, 2)
	for j := 0; j < 128; j++ {
		jmp = j * 8
		copy(tmp2, b.set[jmp+2:jmp+4])
		copy(b.set[jmp+2:jmp+4], b.set[(128*8)+jmp:(128*8)+jmp+2])
		copy(b.set[(128*8)+jmp:(128*8)+jmp+2], tmp2)

		copy(tmp2, b.set[jmp+6:jmp+8])
		copy(b.set[jmp+6:jmp+8], b.set[(128*8)+jmp+4:(128*8)+jmp+6])
		copy(b.set[(128*8)+jmp+4:(128*8)+jmp+6], tmp2)

		copy(tmp2, b.set[(256*8)+jmp+2:(256*8)+jmp+4])
		copy(b.set[(256*8)+jmp+2:(256*8)+jmp+4], b.set[(384*8)+jmp:(384*8)+jmp+2])
		copy(b.set[(384*8)+jmp:(384*8)+jmp+2], tmp2)

		copy(tmp2, b.set[(256*8)+jmp+6:(256*8)+jmp+8])
		copy(b.set[(256*8)+jmp+6:(256*8)+jmp+8], b.set[(384*8)+jmp+4:(384*8)+jmp+6])
		copy(b.set[(384*8)+jmp+4:(384*8)+jmp+6], tmp2)
	}

	// Transpose 1 x 64 blocks
	tmp := make([]uint64, 1)
	for k := 0; k < 64; k++ {
		jmp = k * 8
		copy(tmp, b.set[jmp+1:jmp+2])
		copy(b.set[jmp+1:jmp+2], b.set[(64*8)+jmp:(64*8)+jmp+1])
		copy(b.set[(64*8)+jmp:(64*8)+jmp+1], tmp)

		copy(tmp, b.set[jmp+3:jmp+4])
		copy(b.set[jmp+3:jmp+4], b.set[(64*8)+jmp+2:(64*8)+jmp+3])
		copy(b.set[(64*8)+jmp+2:(64*8)+jmp+3], tmp)

		copy(tmp, b.set[jmp+5:jmp+6])
		copy(b.set[jmp+5:jmp+6], b.set[(64*8)+jmp+4:(64*8)+jmp+5])
		copy(b.set[(64*8)+jmp+4:(64*8)+jmp+5], tmp)

		copy(tmp, b.set[jmp+7:jmp+8])
		copy(b.set[jmp+7:jmp+8], b.set[(64*8)+jmp+6:(64*8)+jmp+7])
		copy(b.set[(64*8)+jmp+6:(64*8)+jmp+7], tmp)

		copy(tmp, b.set[(128*8)+jmp+1:(128*8)+jmp+2])
		copy(b.set[(128*8)+jmp+1:(128*8)+jmp+2], b.set[(192*8)+jmp:(192*8)+jmp+1])
		copy(b.set[(192*8)+jmp:(192*8)+jmp+1], tmp)

		copy(tmp, b.set[(128*8)+jmp+3:(128*8)+jmp+4])
		copy(b.set[(128*8)+jmp+3:(128*8)+jmp+4], b.set[(192*8)+jmp+2:(192*8)+jmp+3])
		copy(b.set[(192*8)+jmp+2:(192*8)+jmp+3], tmp)

		copy(tmp, b.set[(128*8)+jmp+5:(128*8)+jmp+6])
		copy(b.set[(128*8)+jmp+5:(128*8)+jmp+6], b.set[(192*8)+jmp+4:(192*8)+jmp+5])
		copy(b.set[(192*8)+jmp+4:(192*8)+jmp+5], tmp)

		copy(tmp, b.set[(128*8)+jmp+7:(128*8)+jmp+8])
		copy(b.set[(128*8)+jmp+7:(128*8)+jmp+8], b.set[(192*8)+jmp+6:(192*8)+jmp+7])
		copy(b.set[(192*8)+jmp+6:(192*8)+jmp+7], tmp)
		//
		copy(tmp, b.set[(256*8)+jmp+1:(256*8)+jmp+2])
		copy(b.set[(256*8)+jmp+1:(256*8)+jmp+2], b.set[(320*8)+jmp:(320*8)+jmp+1])
		copy(b.set[(320*8)+jmp:(320*8)+jmp+1], tmp)

		copy(tmp, b.set[(256*8)+jmp+3:(256*8)+jmp+4])
		copy(b.set[(256*8)+jmp+3:(256*8)+jmp+4], b.set[(320*8)+jmp+2:(320*8)+jmp+3])
		copy(b.set[(320*8)+jmp+2:(320*8)+jmp+3], tmp)

		copy(tmp, b.set[(256*8)+jmp+5:(256*8)+jmp+6])
		copy(b.set[(256*8)+jmp+5:(256*8)+jmp+6], b.set[(320*8)+jmp+4:(320*8)+jmp+5])
		copy(b.set[(320*8)+jmp+4:(320*8)+jmp+5], tmp)

		copy(tmp, b.set[(256*8)+jmp+7:(256*8)+jmp+8])
		copy(b.set[(256*8)+jmp+7:(256*8)+jmp+8], b.set[(320*8)+jmp+6:(320*8)+jmp+7])
		copy(b.set[(320*8)+jmp+6:(320*8)+jmp+7], tmp)

		copy(tmp, b.set[(384*8)+jmp+1:(384*8)+jmp+2])
		copy(b.set[(384*8)+jmp+1:(384*8)+jmp+2], b.set[(448*8)+jmp:(448*8)+jmp+1])
		copy(b.set[(448*8)+jmp:(448*8)+jmp+1], tmp)

		copy(tmp, b.set[(384*8)+jmp+3:(384*8)+jmp+4])
		copy(b.set[(384*8)+jmp+3:(384*8)+jmp+4], b.set[(448*8)+jmp+2:(448*8)+jmp+3])
		copy(b.set[(448*8)+jmp+2:(448*8)+jmp+3], tmp)

		copy(tmp, b.set[(384*8)+jmp+5:(384*8)+jmp+6])
		copy(b.set[(384*8)+jmp+5:(384*8)+jmp+6], b.set[(448*8)+jmp+4:(448*8)+jmp+5])
		copy(b.set[(448*8)+jmp+4:(448*8)+jmp+5], tmp)

		copy(tmp, b.set[(384*8)+jmp+7:(384*8)+jmp+8])
		copy(b.set[(384*8)+jmp+7:(384*8)+jmp+8], b.set[(448*8)+jmp+6:(448*8)+jmp+7])
		copy(b.set[(448*8)+jmp+6:(448*8)+jmp+7], tmp)

	}

	// Bitwise transposition
	for blk := 0; blk < 8; blk++ {
		for col := 0; col < 8; col++ {
			//transpose64(b, blk, col)
			transpose64(b, blk, col)
		}
	}
}

// swap64 swaps two subsets of rows in a 512x8 bit matrix which is
// held as a contiguous uint64 array in a BitVect.
func swap64(a, b int, vect *BitVect, width int, tmp []uint64) {
	copy(tmp, vect.set[a:a+width])
	copy(vect.set[a:a+width], vect.set[b:b+width])
	copy(vect.set[b:b+width], tmp)
}

// TODO not done -> doesn't seem to substantially improve performance
// unrolledTranspose performs a cache-oblivious, in-place, contiguous transpose.
// Since a BitVect represents a 512 by 512 square bit matrix, transposition will
// be performed blockwise starting with blocks of 256 x 4, swapped about the
// principle diagonal. Then blocks size will decrease by half until it is only
// 64 x 1. The remaining transposition steps are performed using bit masks and
// shifts. Operations are performed on blocks of bits of size 32, 16, 8, 4, 2,
// and 1. Since the input is square, the transposition can be performed in place.
func (b *BitVect) unrolledTranspose() {
	var jmp int
	// Transpose 4 x 256 blocks
	width := 4
	tmp4 := make([]uint64, width)

	swap64((4)+(8*0), (256*8)+(8*0), b, width, tmp4)
	swap64((4)+(8*1), (256*8)+(8*1), b, width, tmp4)
	swap64((4)+(8*2), (256*8)+(8*2), b, width, tmp4)
	swap64((4)+(8*3), (256*8)+(8*3), b, width, tmp4)
	swap64((4)+(8*4), (256*8)+(8*4), b, width, tmp4)
	swap64((4)+(8*5), (256*8)+(8*5), b, width, tmp4)
	swap64((4)+(8*6), (256*8)+(8*6), b, width, tmp4)
	swap64((4)+(8*7), (256*8)+(8*7), b, width, tmp4)
	swap64((4)+(8*8), (256*8)+(8*8), b, width, tmp4)
	swap64((4)+(8*9), (256*8)+(8*9), b, width, tmp4)
	swap64((4)+(8*10), (256*8)+(8*10), b, width, tmp4)
	swap64((4)+(8*11), (256*8)+(8*11), b, width, tmp4)
	swap64((4)+(8*12), (256*8)+(8*12), b, width, tmp4)
	swap64((4)+(8*13), (256*8)+(8*13), b, width, tmp4)
	swap64((4)+(8*14), (256*8)+(8*14), b, width, tmp4)
	swap64((4)+(8*15), (256*8)+(8*15), b, width, tmp4)
	swap64((4)+(8*16), (256*8)+(8*16), b, width, tmp4)
	swap64((4)+(8*17), (256*8)+(8*17), b, width, tmp4)
	swap64((4)+(8*18), (256*8)+(8*18), b, width, tmp4)
	swap64((4)+(8*19), (256*8)+(8*19), b, width, tmp4)
	swap64((4)+(8*20), (256*8)+(8*20), b, width, tmp4)
	swap64((4)+(8*21), (256*8)+(8*21), b, width, tmp4)
	swap64((4)+(8*22), (256*8)+(8*22), b, width, tmp4)
	swap64((4)+(8*23), (256*8)+(8*23), b, width, tmp4)
	swap64((4)+(8*24), (256*8)+(8*24), b, width, tmp4)
	swap64((4)+(8*25), (256*8)+(8*25), b, width, tmp4)
	swap64((4)+(8*26), (256*8)+(8*26), b, width, tmp4)
	swap64((4)+(8*27), (256*8)+(8*27), b, width, tmp4)
	swap64((4)+(8*28), (256*8)+(8*28), b, width, tmp4)
	swap64((4)+(8*29), (256*8)+(8*29), b, width, tmp4)
	swap64((4)+(8*30), (256*8)+(8*30), b, width, tmp4)
	swap64((4)+(8*31), (256*8)+(8*31), b, width, tmp4)
	swap64((4)+(8*32), (256*8)+(8*32), b, width, tmp4)
	swap64((4)+(8*33), (256*8)+(8*33), b, width, tmp4)
	swap64((4)+(8*34), (256*8)+(8*34), b, width, tmp4)
	swap64((4)+(8*35), (256*8)+(8*35), b, width, tmp4)
	swap64((4)+(8*36), (256*8)+(8*36), b, width, tmp4)
	swap64((4)+(8*37), (256*8)+(8*37), b, width, tmp4)
	swap64((4)+(8*38), (256*8)+(8*38), b, width, tmp4)
	swap64((4)+(8*39), (256*8)+(8*39), b, width, tmp4)
	swap64((4)+(8*40), (256*8)+(8*40), b, width, tmp4)
	swap64((4)+(8*41), (256*8)+(8*41), b, width, tmp4)
	swap64((4)+(8*42), (256*8)+(8*42), b, width, tmp4)
	swap64((4)+(8*43), (256*8)+(8*43), b, width, tmp4)
	swap64((4)+(8*44), (256*8)+(8*44), b, width, tmp4)
	swap64((4)+(8*45), (256*8)+(8*45), b, width, tmp4)
	swap64((4)+(8*46), (256*8)+(8*46), b, width, tmp4)
	swap64((4)+(8*47), (256*8)+(8*47), b, width, tmp4)
	swap64((4)+(8*48), (256*8)+(8*48), b, width, tmp4)
	swap64((4)+(8*49), (256*8)+(8*49), b, width, tmp4)
	swap64((4)+(8*50), (256*8)+(8*50), b, width, tmp4)
	swap64((4)+(8*51), (256*8)+(8*51), b, width, tmp4)
	swap64((4)+(8*52), (256*8)+(8*52), b, width, tmp4)
	swap64((4)+(8*53), (256*8)+(8*53), b, width, tmp4)
	swap64((4)+(8*54), (256*8)+(8*54), b, width, tmp4)
	swap64((4)+(8*55), (256*8)+(8*55), b, width, tmp4)
	swap64((4)+(8*56), (256*8)+(8*56), b, width, tmp4)
	swap64((4)+(8*57), (256*8)+(8*57), b, width, tmp4)
	swap64((4)+(8*58), (256*8)+(8*58), b, width, tmp4)
	swap64((4)+(8*59), (256*8)+(8*59), b, width, tmp4)
	swap64((4)+(8*60), (256*8)+(8*60), b, width, tmp4)
	swap64((4)+(8*61), (256*8)+(8*61), b, width, tmp4)
	swap64((4)+(8*62), (256*8)+(8*61), b, width, tmp4)
	swap64((4)+(8*63), (256*8)+(8*62), b, width, tmp4)
	swap64((4)+(8*64), (256*8)+(8*64), b, width, tmp4)
	swap64((4)+(8*65), (256*8)+(8*65), b, width, tmp4)
	swap64((4)+(8*66), (256*8)+(8*66), b, width, tmp4)
	swap64((4)+(8*67), (256*8)+(8*67), b, width, tmp4)
	swap64((4)+(8*68), (256*8)+(8*68), b, width, tmp4)
	swap64((4)+(8*69), (256*8)+(8*69), b, width, tmp4)
	swap64((4)+(8*70), (256*8)+(8*70), b, width, tmp4)
	swap64((4)+(8*71), (256*8)+(8*71), b, width, tmp4)
	swap64((4)+(8*72), (256*8)+(8*72), b, width, tmp4)
	swap64((4)+(8*73), (256*8)+(8*73), b, width, tmp4)
	swap64((4)+(8*74), (256*8)+(8*74), b, width, tmp4)
	swap64((4)+(8*75), (256*8)+(8*75), b, width, tmp4)
	swap64((4)+(8*76), (256*8)+(8*76), b, width, tmp4)
	swap64((4)+(8*77), (256*8)+(8*77), b, width, tmp4)
	swap64((4)+(8*78), (256*8)+(8*78), b, width, tmp4)
	swap64((4)+(8*79), (256*8)+(8*79), b, width, tmp4)
	swap64((4)+(8*80), (256*8)+(8*80), b, width, tmp4)
	swap64((4)+(8*81), (256*8)+(8*81), b, width, tmp4)
	swap64((4)+(8*82), (256*8)+(8*82), b, width, tmp4)
	swap64((4)+(8*83), (256*8)+(8*83), b, width, tmp4)
	swap64((4)+(8*84), (256*8)+(8*84), b, width, tmp4)
	swap64((4)+(8*85), (256*8)+(8*85), b, width, tmp4)
	swap64((4)+(8*86), (256*8)+(8*86), b, width, tmp4)
	swap64((4)+(8*87), (256*8)+(8*87), b, width, tmp4)
	swap64((4)+(8*88), (256*8)+(8*88), b, width, tmp4)
	swap64((4)+(8*89), (256*8)+(8*89), b, width, tmp4)
	swap64((4)+(8*90), (256*8)+(8*90), b, width, tmp4)
	swap64((4)+(8*91), (256*8)+(8*91), b, width, tmp4)
	swap64((4)+(8*92), (256*8)+(8*92), b, width, tmp4)
	swap64((4)+(8*93), (256*8)+(8*93), b, width, tmp4)
	swap64((4)+(8*94), (256*8)+(8*94), b, width, tmp4)
	swap64((4)+(8*95), (256*8)+(8*95), b, width, tmp4)
	swap64((4)+(8*96), (256*8)+(8*96), b, width, tmp4)
	swap64((4)+(8*97), (256*8)+(8*97), b, width, tmp4)
	swap64((4)+(8*98), (256*8)+(8*98), b, width, tmp4)
	swap64((4)+(8*99), (256*8)+(8*99), b, width, tmp4)
	swap64((4)+(8*100), (256*8)+(8*100), b, width, tmp4)
	swap64((4)+(8*101), (256*8)+(8*101), b, width, tmp4)
	swap64((4)+(8*102), (256*8)+(8*102), b, width, tmp4)
	swap64((4)+(8*103), (256*8)+(8*103), b, width, tmp4)
	swap64((4)+(8*104), (256*8)+(8*104), b, width, tmp4)
	swap64((4)+(8*105), (256*8)+(8*105), b, width, tmp4)
	swap64((4)+(8*106), (256*8)+(8*106), b, width, tmp4)
	swap64((4)+(8*107), (256*8)+(8*107), b, width, tmp4)
	swap64((4)+(8*108), (256*8)+(8*108), b, width, tmp4)
	swap64((4)+(8*109), (256*8)+(8*109), b, width, tmp4)
	swap64((4)+(8*110), (256*8)+(8*110), b, width, tmp4)
	swap64((4)+(8*111), (256*8)+(8*111), b, width, tmp4)
	swap64((4)+(8*112), (256*8)+(8*112), b, width, tmp4)
	swap64((4)+(8*113), (256*8)+(8*113), b, width, tmp4)
	swap64((4)+(8*114), (256*8)+(8*114), b, width, tmp4)
	swap64((4)+(8*115), (256*8)+(8*115), b, width, tmp4)
	swap64((4)+(8*116), (256*8)+(8*116), b, width, tmp4)
	swap64((4)+(8*117), (256*8)+(8*117), b, width, tmp4)
	swap64((4)+(8*118), (256*8)+(8*118), b, width, tmp4)
	swap64((4)+(8*119), (256*8)+(8*119), b, width, tmp4)
	swap64((4)+(8*120), (256*8)+(8*120), b, width, tmp4)
	swap64((4)+(8*121), (256*8)+(8*121), b, width, tmp4)
	swap64((4)+(8*122), (256*8)+(8*122), b, width, tmp4)
	swap64((4)+(8*123), (256*8)+(8*123), b, width, tmp4)
	swap64((4)+(8*124), (256*8)+(8*124), b, width, tmp4)
	swap64((4)+(8*125), (256*8)+(8*125), b, width, tmp4)
	swap64((4)+(8*126), (256*8)+(8*126), b, width, tmp4)
	swap64((4)+(8*127), (256*8)+(8*127), b, width, tmp4)
	swap64((4)+(8*128), (256*8)+(8*128), b, width, tmp4)
	swap64((4)+(8*129), (256*8)+(8*129), b, width, tmp4)
	swap64((4)+(8*130), (256*8)+(8*130), b, width, tmp4)
	swap64((4)+(8*131), (256*8)+(8*131), b, width, tmp4)
	swap64((4)+(8*132), (256*8)+(8*132), b, width, tmp4)
	swap64((4)+(8*133), (256*8)+(8*133), b, width, tmp4)
	swap64((4)+(8*134), (256*8)+(8*134), b, width, tmp4)
	swap64((4)+(8*135), (256*8)+(8*135), b, width, tmp4)
	swap64((4)+(8*136), (256*8)+(8*136), b, width, tmp4)
	swap64((4)+(8*137), (256*8)+(8*137), b, width, tmp4)
	swap64((4)+(8*138), (256*8)+(8*138), b, width, tmp4)
	swap64((4)+(8*139), (256*8)+(8*139), b, width, tmp4)
	swap64((4)+(8*140), (256*8)+(8*140), b, width, tmp4)
	swap64((4)+(8*141), (256*8)+(8*141), b, width, tmp4)
	swap64((4)+(8*142), (256*8)+(8*142), b, width, tmp4)
	swap64((4)+(8*143), (256*8)+(8*143), b, width, tmp4)
	swap64((4)+(8*144), (256*8)+(8*144), b, width, tmp4)
	swap64((4)+(8*145), (256*8)+(8*145), b, width, tmp4)
	swap64((4)+(8*146), (256*8)+(8*146), b, width, tmp4)
	swap64((4)+(8*147), (256*8)+(8*147), b, width, tmp4)
	swap64((4)+(8*148), (256*8)+(8*148), b, width, tmp4)
	swap64((4)+(8*149), (256*8)+(8*149), b, width, tmp4)
	swap64((4)+(8*150), (256*8)+(8*150), b, width, tmp4)
	swap64((4)+(8*151), (256*8)+(8*151), b, width, tmp4)
	swap64((4)+(8*152), (256*8)+(8*152), b, width, tmp4)
	swap64((4)+(8*153), (256*8)+(8*153), b, width, tmp4)
	swap64((4)+(8*154), (256*8)+(8*154), b, width, tmp4)
	swap64((4)+(8*155), (256*8)+(8*155), b, width, tmp4)
	swap64((4)+(8*156), (256*8)+(8*156), b, width, tmp4)
	swap64((4)+(8*157), (256*8)+(8*157), b, width, tmp4)
	swap64((4)+(8*158), (256*8)+(8*158), b, width, tmp4)
	swap64((4)+(8*159), (256*8)+(8*159), b, width, tmp4)
	swap64((4)+(8*160), (256*8)+(8*160), b, width, tmp4)
	swap64((4)+(8*161), (256*8)+(8*161), b, width, tmp4)
	swap64((4)+(8*162), (256*8)+(8*161), b, width, tmp4)
	swap64((4)+(8*163), (256*8)+(8*162), b, width, tmp4)
	swap64((4)+(8*164), (256*8)+(8*164), b, width, tmp4)
	swap64((4)+(8*165), (256*8)+(8*165), b, width, tmp4)
	swap64((4)+(8*166), (256*8)+(8*166), b, width, tmp4)
	swap64((4)+(8*167), (256*8)+(8*167), b, width, tmp4)
	swap64((4)+(8*168), (256*8)+(8*168), b, width, tmp4)
	swap64((4)+(8*169), (256*8)+(8*169), b, width, tmp4)
	swap64((4)+(8*170), (256*8)+(8*170), b, width, tmp4)
	swap64((4)+(8*171), (256*8)+(8*171), b, width, tmp4)
	swap64((4)+(8*172), (256*8)+(8*172), b, width, tmp4)
	swap64((4)+(8*173), (256*8)+(8*173), b, width, tmp4)
	swap64((4)+(8*174), (256*8)+(8*174), b, width, tmp4)
	swap64((4)+(8*175), (256*8)+(8*175), b, width, tmp4)
	swap64((4)+(8*176), (256*8)+(8*176), b, width, tmp4)
	swap64((4)+(8*177), (256*8)+(8*177), b, width, tmp4)
	swap64((4)+(8*178), (256*8)+(8*178), b, width, tmp4)
	swap64((4)+(8*179), (256*8)+(8*179), b, width, tmp4)
	swap64((4)+(8*180), (256*8)+(8*180), b, width, tmp4)
	swap64((4)+(8*181), (256*8)+(8*181), b, width, tmp4)
	swap64((4)+(8*182), (256*8)+(8*182), b, width, tmp4)
	swap64((4)+(8*183), (256*8)+(8*183), b, width, tmp4)
	swap64((4)+(8*184), (256*8)+(8*184), b, width, tmp4)
	swap64((4)+(8*185), (256*8)+(8*185), b, width, tmp4)
	swap64((4)+(8*186), (256*8)+(8*186), b, width, tmp4)
	swap64((4)+(8*187), (256*8)+(8*187), b, width, tmp4)
	swap64((4)+(8*188), (256*8)+(8*188), b, width, tmp4)
	swap64((4)+(8*189), (256*8)+(8*189), b, width, tmp4)
	swap64((4)+(8*190), (256*8)+(8*190), b, width, tmp4)
	swap64((4)+(8*191), (256*8)+(8*191), b, width, tmp4)
	swap64((4)+(8*192), (256*8)+(8*192), b, width, tmp4)
	swap64((4)+(8*193), (256*8)+(8*193), b, width, tmp4)
	swap64((4)+(8*194), (256*8)+(8*194), b, width, tmp4)
	swap64((4)+(8*195), (256*8)+(8*195), b, width, tmp4)
	swap64((4)+(8*196), (256*8)+(8*196), b, width, tmp4)
	swap64((4)+(8*197), (256*8)+(8*197), b, width, tmp4)
	swap64((4)+(8*198), (256*8)+(8*198), b, width, tmp4)
	swap64((4)+(8*199), (256*8)+(8*199), b, width, tmp4)
	swap64((4)+(8*200), (256*8)+(8*200), b, width, tmp4)
	swap64((4)+(8*201), (256*8)+(8*201), b, width, tmp4)
	swap64((4)+(8*202), (256*8)+(8*202), b, width, tmp4)
	swap64((4)+(8*203), (256*8)+(8*203), b, width, tmp4)
	swap64((4)+(8*204), (256*8)+(8*204), b, width, tmp4)
	swap64((4)+(8*205), (256*8)+(8*205), b, width, tmp4)
	swap64((4)+(8*206), (256*8)+(8*206), b, width, tmp4)
	swap64((4)+(8*207), (256*8)+(8*207), b, width, tmp4)
	swap64((4)+(8*208), (256*8)+(8*208), b, width, tmp4)
	swap64((4)+(8*209), (256*8)+(8*209), b, width, tmp4)
	swap64((4)+(8*210), (256*8)+(8*210), b, width, tmp4)
	swap64((4)+(8*211), (256*8)+(8*211), b, width, tmp4)
	swap64((4)+(8*212), (256*8)+(8*212), b, width, tmp4)
	swap64((4)+(8*213), (256*8)+(8*213), b, width, tmp4)
	swap64((4)+(8*214), (256*8)+(8*214), b, width, tmp4)
	swap64((4)+(8*215), (256*8)+(8*215), b, width, tmp4)
	swap64((4)+(8*216), (256*8)+(8*216), b, width, tmp4)
	swap64((4)+(8*217), (256*8)+(8*217), b, width, tmp4)
	swap64((4)+(8*218), (256*8)+(8*218), b, width, tmp4)
	swap64((4)+(8*219), (256*8)+(8*219), b, width, tmp4)
	swap64((4)+(8*220), (256*8)+(8*220), b, width, tmp4)
	swap64((4)+(8*221), (256*8)+(8*221), b, width, tmp4)
	swap64((4)+(8*222), (256*8)+(8*222), b, width, tmp4)
	swap64((4)+(8*223), (256*8)+(8*223), b, width, tmp4)
	swap64((4)+(8*224), (256*8)+(8*224), b, width, tmp4)
	swap64((4)+(8*225), (256*8)+(8*225), b, width, tmp4)
	swap64((4)+(8*226), (256*8)+(8*226), b, width, tmp4)
	swap64((4)+(8*227), (256*8)+(8*227), b, width, tmp4)
	swap64((4)+(8*228), (256*8)+(8*228), b, width, tmp4)
	swap64((4)+(8*229), (256*8)+(8*229), b, width, tmp4)
	swap64((4)+(8*230), (256*8)+(8*230), b, width, tmp4)
	swap64((4)+(8*231), (256*8)+(8*231), b, width, tmp4)
	swap64((4)+(8*232), (256*8)+(8*232), b, width, tmp4)
	swap64((4)+(8*233), (256*8)+(8*233), b, width, tmp4)
	swap64((4)+(8*234), (256*8)+(8*234), b, width, tmp4)
	swap64((4)+(8*235), (256*8)+(8*235), b, width, tmp4)
	swap64((4)+(8*236), (256*8)+(8*236), b, width, tmp4)
	swap64((4)+(8*237), (256*8)+(8*237), b, width, tmp4)
	swap64((4)+(8*238), (256*8)+(8*238), b, width, tmp4)
	swap64((4)+(8*239), (256*8)+(8*239), b, width, tmp4)
	swap64((4)+(8*240), (256*8)+(8*240), b, width, tmp4)
	swap64((4)+(8*241), (256*8)+(8*241), b, width, tmp4)
	swap64((4)+(8*242), (256*8)+(8*242), b, width, tmp4)
	swap64((4)+(8*243), (256*8)+(8*243), b, width, tmp4)
	swap64((4)+(8*244), (256*8)+(8*244), b, width, tmp4)
	swap64((4)+(8*245), (256*8)+(8*245), b, width, tmp4)
	swap64((4)+(8*246), (256*8)+(8*246), b, width, tmp4)
	swap64((4)+(8*247), (256*8)+(8*247), b, width, tmp4)
	swap64((4)+(8*248), (256*8)+(8*248), b, width, tmp4)
	swap64((4)+(8*249), (256*8)+(8*249), b, width, tmp4)
	swap64((4)+(8*250), (256*8)+(8*250), b, width, tmp4)
	swap64((4)+(8*251), (256*8)+(8*251), b, width, tmp4)
	swap64((4)+(8*252), (256*8)+(8*252), b, width, tmp4)
	swap64((4)+(8*253), (256*8)+(8*253), b, width, tmp4)
	swap64((4)+(8*254), (256*8)+(8*254), b, width, tmp4)
	swap64((4)+(8*255), (256*8)+(8*255), b, width, tmp4)

	// Transpose 2 x 128 blocks
	tmp2 := make([]uint64, 2)
	for j := 0; j < 128; j++ {
		jmp = j * 8
		copy(tmp2, b.set[jmp+2:jmp+4])
		copy(b.set[jmp+2:jmp+4], b.set[(128*8)+jmp:(128*8)+jmp+2])
		copy(b.set[(128*8)+jmp:(128*8)+jmp+2], tmp2)

		copy(tmp2, b.set[jmp+6:jmp+8])
		copy(b.set[jmp+6:jmp+8], b.set[(128*8)+jmp+4:(128*8)+jmp+6])
		copy(b.set[(128*8)+jmp+4:(128*8)+jmp+6], tmp2)

		copy(tmp2, b.set[(256*8)+jmp+2:(256*8)+jmp+4])
		copy(b.set[(256*8)+jmp+2:(256*8)+jmp+4], b.set[(384*8)+jmp:(384*8)+jmp+2])
		copy(b.set[(384*8)+jmp:(384*8)+jmp+2], tmp2)

		copy(tmp2, b.set[(256*8)+jmp+6:(256*8)+jmp+8])
		copy(b.set[(256*8)+jmp+6:(256*8)+jmp+8], b.set[(384*8)+jmp+4:(384*8)+jmp+6])
		copy(b.set[(384*8)+jmp+4:(384*8)+jmp+6], tmp2)
	}

	// Transpose 1 x 64 blocks
	tmp := make([]uint64, 1)
	for k := 0; k < 64; k++ {
		jmp = k * 8
		copy(tmp, b.set[jmp+1:jmp+2])
		copy(b.set[jmp+1:jmp+2], b.set[(64*8)+jmp:(64*8)+jmp+1])
		copy(b.set[(64*8)+jmp:(64*8)+jmp+1], tmp)

		copy(tmp, b.set[jmp+3:jmp+4])
		copy(b.set[jmp+3:jmp+4], b.set[(64*8)+jmp+2:(64*8)+jmp+3])
		copy(b.set[(64*8)+jmp+2:(64*8)+jmp+3], tmp)

		copy(tmp, b.set[jmp+5:jmp+6])
		copy(b.set[jmp+5:jmp+6], b.set[(64*8)+jmp+4:(64*8)+jmp+5])
		copy(b.set[(64*8)+jmp+4:(64*8)+jmp+5], tmp)

		copy(tmp, b.set[jmp+7:jmp+8])
		copy(b.set[jmp+7:jmp+8], b.set[(64*8)+jmp+6:(64*8)+jmp+7])
		copy(b.set[(64*8)+jmp+6:(64*8)+jmp+7], tmp)

		copy(tmp, b.set[(128*8)+jmp+1:(128*8)+jmp+2])
		copy(b.set[(128*8)+jmp+1:(128*8)+jmp+2], b.set[(192*8)+jmp:(192*8)+jmp+1])
		copy(b.set[(192*8)+jmp:(192*8)+jmp+1], tmp)

		copy(tmp, b.set[(128*8)+jmp+3:(128*8)+jmp+4])
		copy(b.set[(128*8)+jmp+3:(128*8)+jmp+4], b.set[(192*8)+jmp+2:(192*8)+jmp+3])
		copy(b.set[(192*8)+jmp+2:(192*8)+jmp+3], tmp)

		copy(tmp, b.set[(128*8)+jmp+5:(128*8)+jmp+6])
		copy(b.set[(128*8)+jmp+5:(128*8)+jmp+6], b.set[(192*8)+jmp+4:(192*8)+jmp+5])
		copy(b.set[(192*8)+jmp+4:(192*8)+jmp+5], tmp)

		copy(tmp, b.set[(128*8)+jmp+7:(128*8)+jmp+8])
		copy(b.set[(128*8)+jmp+7:(128*8)+jmp+8], b.set[(192*8)+jmp+6:(192*8)+jmp+7])
		copy(b.set[(192*8)+jmp+6:(192*8)+jmp+7], tmp)
		//
		copy(tmp, b.set[(256*8)+jmp+1:(256*8)+jmp+2])
		copy(b.set[(256*8)+jmp+1:(256*8)+jmp+2], b.set[(320*8)+jmp:(320*8)+jmp+1])
		copy(b.set[(320*8)+jmp:(320*8)+jmp+1], tmp)

		copy(tmp, b.set[(256*8)+jmp+3:(256*8)+jmp+4])
		copy(b.set[(256*8)+jmp+3:(256*8)+jmp+4], b.set[(320*8)+jmp+2:(320*8)+jmp+3])
		copy(b.set[(320*8)+jmp+2:(320*8)+jmp+3], tmp)

		copy(tmp, b.set[(256*8)+jmp+5:(256*8)+jmp+6])
		copy(b.set[(256*8)+jmp+5:(256*8)+jmp+6], b.set[(320*8)+jmp+4:(320*8)+jmp+5])
		copy(b.set[(320*8)+jmp+4:(320*8)+jmp+5], tmp)

		copy(tmp, b.set[(256*8)+jmp+7:(256*8)+jmp+8])
		copy(b.set[(256*8)+jmp+7:(256*8)+jmp+8], b.set[(320*8)+jmp+6:(320*8)+jmp+7])
		copy(b.set[(320*8)+jmp+6:(320*8)+jmp+7], tmp)

		copy(tmp, b.set[(384*8)+jmp+1:(384*8)+jmp+2])
		copy(b.set[(384*8)+jmp+1:(384*8)+jmp+2], b.set[(448*8)+jmp:(448*8)+jmp+1])
		copy(b.set[(448*8)+jmp:(448*8)+jmp+1], tmp)

		copy(tmp, b.set[(384*8)+jmp+3:(384*8)+jmp+4])
		copy(b.set[(384*8)+jmp+3:(384*8)+jmp+4], b.set[(448*8)+jmp+2:(448*8)+jmp+3])
		copy(b.set[(448*8)+jmp+2:(448*8)+jmp+3], tmp)

		copy(tmp, b.set[(384*8)+jmp+5:(384*8)+jmp+6])
		copy(b.set[(384*8)+jmp+5:(384*8)+jmp+6], b.set[(448*8)+jmp+4:(448*8)+jmp+5])
		copy(b.set[(448*8)+jmp+4:(448*8)+jmp+5], tmp)

		copy(tmp, b.set[(384*8)+jmp+7:(384*8)+jmp+8])
		copy(b.set[(384*8)+jmp+7:(384*8)+jmp+8], b.set[(448*8)+jmp+6:(448*8)+jmp+7])
		copy(b.set[(448*8)+jmp+6:(448*8)+jmp+7], tmp)

	}

	// Bitwise transposition
	for blk := 0; blk < 8; blk++ {
		for col := 0; col < 8; col++ {
			//transpose64(b, blk, col)
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
