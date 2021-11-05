package util

import (
	"math/rand"
	"runtime"
	"testing"
)

var (
	nmsg        = 1024
	nworkers    = runtime.NumCPU()
	byteBlock   = sampleRandomTall(prng, nmsg)
	randomBlock = unravelTall(byteBlock, 0)
)

func genTranBlock() BitVect {
	tranBlock2D := make([][]byte, 512)
	for row := range tranBlock2D {
		tranBlock2D[row] = make([]byte, 64)
		for c := 0; c < 64; c++ {
			tranBlock2D[row][c] = 0b01010101
		}
	}
	return unravelTall(tranBlock2D, 0)
}

// sampleRandomTall fills an m by 64 byte matrix (512 bits wide) with
// pseudorandom bytes.
func sampleRandomTall(r *rand.Rand, m int) [][]byte {
	// instantiate matrix
	matrix := make([][]byte, m)

	for row := range matrix {
		matrix[row] = make([]byte, 64)
		r.Read(matrix[row])
	}

	return matrix
}

// sampleRandomWide fills a 512 by n byte matrix (512 bits tall) with
// pseudorandom bytes.
func sampleRandomWide(r *rand.Rand, n int) [][]byte {
	// instantiate matrix
	matrix := make([][]byte, 512)

	for row := range matrix {
		matrix[row] = make([]byte, n)
		r.Read(matrix[row])
	}

	return matrix
}

func TestUnReRavelingTall(t *testing.T) {
	trange := []int{512, 512 * 2, 512 * 3, 512 * 4}
	for _, a := range trange {
		matrix := sampleRandomTall(prng, a)
		// TALL m x 64
		// determine number of blocks to split original matrix
		nblks := len(matrix) / 512

		rerav := make([][]byte, len(matrix))
		for r := range rerav {
			rerav[r] = make([]byte, len(matrix[0]))
		}

		for id := 0; id < nblks; id++ {
			b := unravelTall(matrix, id)
			b.ravelToTall(rerav, id)
		}

		// check
		for k := range rerav {
			for l := range rerav[k] {
				if rerav[k][l] != matrix[k][l] {
					t.Fatal("Unraveled and reraveled tall (", a, ") matrix did not match with original at row", k, ".")
				}
			}
		}
	}
}

func TestUnReRavelingWide(t *testing.T) {
	trange := []int{64, 128, 512}
	for _, a := range trange {
		matrix := sampleRandomWide(prng, a)
		// WIDE 512 x n
		// determine number of blocks to split original matrix
		nblks := len(matrix[0]) / 64

		trans := make([][]byte, len(matrix))
		for r := range trans {
			trans[r] = make([]byte, len(matrix[0]))
		}

		for id := 0; id < nblks; id++ {
			b := unravelWide(matrix, id)
			b.ravelToWide(trans, id)
		}

		// check
		for k := range trans {
			for l := range trans[k] {
				if trans[k][l] != matrix[k][l] {
					t.Fatal("Unraveled and reraveled wide (", a, ") matrix did not match with original at row", k, ".")
				}
			}
		}
	}
}

func TestTranspose512x512(t *testing.T) {
	tr := randomBlock

	tr.transpose()
	tr.transpose()
	// check if transpose is correct
	if tr != randomBlock {
		t.Fatalf("Block incorrectly transposed.")
	}
	/* TODO - CheckTranspose not working
	if !tr.CheckTranspose(randomBlock) {
		b.Fatalf("Block incorrectly transposed.")
	}
	*/
}

func TestIfLittleEndianTranspose(t *testing.T) {
	tr := genTranBlock()
	// 0101....
	// 0101....
	// 0101....
	//tr.printBits(64)
	tr.transpose()
	//tr.printBits(64)
	// If Little Endian, we expect the resulting rows to be
	// 1111....
	// 0000....
	// 1111....

	// check if Little Endian
	for i := 0; i < 512; i++ {
		if i%2 == 1 { // odd
			if tr.set[i*8] != 0 {
				t.Fatalf("transpose appears to be Big Endian")
			}
		} else {
			if tr.set[i*8] != 0xFFFFFFFFFFFFFFFF {
				t.Fatalf("transpose appears to be Big Endian")
			}
		}
	}
}

func TestConcurrentTranspose(t *testing.T) {
	// TALL
	trange := []int{512, 512 * 2, 512 * 3, 512 * 4}
	for _, m := range trange {
		orig := sampleRandomTall(prng, m)
		tr := ConcurrentTranspose(orig, nworkers)
		dtr := ConcurrentTranspose(tr, nworkers)
		// test
		for k := range orig {
			for l := range orig[k] {
				// note the weird aerobics we have to do here because of the residual insignificant rows added
				// due to the encoding of 8 rows in one column element.
				if orig[k][l] != dtr[len(dtr)-len(orig)+k][l] {
					t.Fatal("Doubly-transposed tall (", m, ") matrix did not match with original at row", k, ".")
				}
			}
		}
	}
	// WIDE
	//trange = []int{64, 64 * 2, 64 * 3, 64 * 4}
	trange = []int{64}
	for _, m := range trange {
		orig := sampleRandomWide(prng, m)
		tr := ConcurrentTranspose(orig, nworkers)
		dtr := ConcurrentTranspose(tr, nworkers)
		//test
		for k := range dtr {
			for l := range dtr[k] {
				if dtr[k][l] != orig[k][l] {
					t.Fatal("Doubly-transposed wide (", m, ") matrix did not match with original at row", k, ".")
				}
			}
		}
	}
}

func BenchmarkTranspose512x512(b *testing.B) {
	tr := randomBlock
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.transpose()
	}
}

// Test transpose with the added overhead of creating the blocks
// and writing to an output transpose matrix.
func BenchmarkTranspose(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConcurrentTranspose(byteBlock, 1)
	}
}

func BenchmarkConcurrentTranspose(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConcurrentTranspose(byteBlock, nworkers)
	}
}
