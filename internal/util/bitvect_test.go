package util

import (
	"crypto/rand"
	"runtime"
	"testing"
)

var (
	nmsg     = 1024
	nworkers = runtime.GOMAXPROCS(0)
)

// genTranBlock creates a 512x512 bit block where every bit position
// alternates between 0 and 1. When transposed, this block should
// consists of rows of all 0s alternating with rows of all 1s.
func genZebraBlock() BitVect {
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
func sampleRandomTall(m int) [][]byte {
	// instantiate matrix
	matrix := make([][]byte, m)

	for row := range matrix {
		matrix[row] = make([]byte, 64)
		rand.Read(matrix[row])
	}

	return matrix
}

// sampleRandomWide fills a 512 by n byte matrix (512 bits tall) with
// pseudorandom bytes.
func sampleRandomWide(n int) [][]byte {
	// instantiate matrix
	matrix := make([][]byte, 512)

	for row := range matrix {
		matrix[row] = make([]byte, n)
		rand.Read(matrix[row])
	}

	return matrix
}

func TestUnReRavelingTall(t *testing.T) {
	trange := []int{512, 512 * 2, 512 * 3, 512 * 4}
	for _, a := range trange {
		matrix := sampleRandomTall(a)
		// determine number of blocks to split original matrix (m x 64)
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
		matrix := sampleRandomWide(a)
		// determine number of blocks to split original matrix (512 x n)
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

// Test single block tranposition
func TestTranspose512x512(t *testing.T) {
	tr := unravelTall(sampleRandomTall(nmsg), 0)
	orig := BitVect{tr.set} // copy to check after

	tr.transpose()
	tr.transpose()
	// check if transpose is correct
	if tr != orig {
		t.Fatalf("Block incorrectly transposed.")
	}
}

func TestIfLittleEndianTranspose(t *testing.T) {
	tr := genZebraBlock()
	//tr.printBits(64)
	// 0101....
	// 0101....
	// 0101....
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

func TestConcurrentTransposeTall(t *testing.T) {
	trange := []int{512, 512 * 2, 512 * 3, 512 * 4}
	for _, m := range trange {
		orig := sampleRandomTall(m)
		tr := ConcurrentTransposeTall(orig)
		dtr := ConcurrentTransposeWide(tr, nworkers)
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
}

func TestConcurrentTransposeWide(t *testing.T) {
	trange := []int{64, 64 * 2, 64 * 3, 64 * 4}
	for _, m := range trange {
		orig := sampleRandomWide(m)
		tr := ConcurrentTransposeWide(orig, nworkers)
		dtr := ConcurrentTransposeTall(tr)
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

// BenchmarkTranspose512x512 benchmarks just transposing a single
// BitVect block.
func BenchmarkTranspose512x512(b *testing.B) {
	tr := unravelTall(sampleRandomTall(nmsg), 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.transpose()
	}
}

// BenchmarkTranspose tests the BitVect transpose with the overhead
// of having to pull the blocks out of a larger matrix and write to
// a new tranposed matrix. In this case, we limit it to a single thread.
func BenchmarkTranspose(b *testing.B) {
	byteBlock := sampleRandomTall(nmsg)
	runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentTransposeTall(byteBlock)
	}
}

// BenchmarkConcurrentTranspose is the same as BenchmarkTranspose but
// we allow a number of threads equal to the GOMAXPROCS.
func BenchmarkConcurrentTranspose(b *testing.B) {
	byteBlock := sampleRandomTall(nmsg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentTransposeTall(byteBlock)
	}
}
