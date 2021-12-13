// +build !amd64

package util

import "encoding/binary"

// unravelTall populates a BitVect from a 2D matrix of bytes. The matrix
// must have 64 columns and a multiple of 512 rows. idx is the block target.
func (b *BitVect) unravelTall(matrix [][]byte, idx int) {
	for i := 0; i < 512; i++ {
		for j := 0; j < 8; j++ {
			b.set[(i*8)+j] = binary.LittleEndian.Uint64(matrix[(512*idx)+i][j*8 : (j+1)*8])
		}
	}
}

// unravelWide populates a BitVect from a 2D matrix of bytes. The matrix
// must have a multiple of 64 columns and 512 rows. idx is the block target.
func (b *BitVect) unravelWide(matrix [][]byte, idx int) {
	for i := 0; i < 512; i++ {
		for j := 0; j < 8; j++ {
			b.set[(i*8)+j] = binary.LittleEndian.Uint64(matrix[i][(idx*64)+(j*8) : (idx*64)+((j+1)*8)])
		}
	}
}

// ravelToTall reconstructs a subsection of a tall (mx64) matrix from a BitVect.
func (b *BitVect) ravelToTall(matrix [][]byte, idx int) {
	for i := 0; i < 512; i++ {
		for j := 0; j < 8; j++ {
			binary.LittleEndian.PutUint64(matrix[(idx*512)+i][j*8:(j+1)*8], b.set[(i*8)+j])
		}
	}
}

// ravelToWide reconstructs a subsection of a wide (512xn) matrix from a BitVect.
func (b *BitVect) ravelToWide(matrix [][]byte, idx int) {
	for i := 0; i < 512; i++ {
		for j := 0; j < 8; j++ {
			binary.LittleEndian.PutUint64(matrix[i][(idx*64)+(j*8):(idx*64)+((j+1)*8)], b.set[(i*8)+j])
		}
	}
}
