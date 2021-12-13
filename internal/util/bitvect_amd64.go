// +build amd64

package util

import (
	"github.com/alecthomas/unsafeslice"
)

// unravelTall populates a BitVect from a 2D matrix of bytes. The matrix
// must have 64 columns and a multiple of 512 rows. idx is the block target.
// Only tested on x86-64.
func (b *BitVect) unravelTall(matrix [][]byte, idx int) {
	for i := 0; i < 512; i++ {
		copy(b.set[(i)*8:(i+1)*8], unsafeslice.Uint64SliceFromByteSlice(matrix[(512*idx)+i]))
	}
}

// unravelWide populates a BitVect from a 2D matrix of bytes. The matrix
// must have a multiple of 64 columns and 512 rows. idx is the block target.
// Only tested on x86-64.
func (b *BitVect) unravelWide(matrix [][]byte, idx int) {
	for i := 0; i < 512; i++ {
		copy(b.set[i*8:(i+1)*8], unsafeslice.Uint64SliceFromByteSlice(matrix[i][idx*64:(64*idx)+64]))
	}
}

// ravelToTall reconstructs a subsection of a tall (mx64) matrix from a BitVect.
// Only tested on x86-64.
func (b *BitVect) ravelToTall(matrix [][]byte, idx int) {
	for i := 0; i < 512; i++ {
		copy(matrix[(idx*512)+i][:], unsafeslice.ByteSliceFromUint64Slice(b.set[i*8:(i+1)*8]))
	}
}

// ravelToWide reconstructs a subsection of a wide (512xn) matrix from a BitVect.
// Only tested on x86-64.
func (b *BitVect) ravelToWide(matrix [][]byte, idx int) {
	for i := 0; i < 512; i++ {
		copy(matrix[i][idx*64:(idx+1)*64], unsafeslice.ByteSliceFromUint64Slice(b.set[(i*8):(i+1)*8]))
	}
}
