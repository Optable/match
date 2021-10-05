package util

import (
	"testing"
)

var nmsg = 1024
var nworkers = 6
var uintBlock = SampleRandomTall(prng, nmsg)
var randomBlock = unravelTall(uintBlock, 0)
var (
	oneMil         = 1000000
	fiveMil        = 5000000
	tenMil         = 10000000
	thirtyMil      = 30000000
	fiftyMil       = 50000000
	eightyMil      = 80000000
	oneHundredMil  = 100000000
	fiveHundredMil = 500000000
	oneBil         = 1000000000
)

func genOrigBlock() BitVect {
	origBlock2D := make([][]uint64, 512)
	for row := range origBlock2D {
		origBlock2D[row] = make([]uint64, 8)
		for c := 0; c < 8; c++ {
			// alternating rows of all 0s and all 1s (bits)
			if row%2 == 1 {
				origBlock2D[row][c] = ^uint64(0)
			}
		}
	}
	return unravelTall(origBlock2D, 0)
}

func genTranBlock() BitVect {
	tranBlock2D := make([][]uint64, 512)
	for row := range tranBlock2D {
		tranBlock2D[row] = make([]uint64, 8)
		for c := 0; c < 8; c++ {
			tranBlock2D[row][c] = 0x5555555555555555 // 01010...01
		}
	}
	return unravelTall(tranBlock2D, 0)
}

func genOnesBlock() BitVect {
	onesBlock2D := make([][]uint64, 512)
	for row := range onesBlock2D {
		onesBlock2D[row] = make([]uint64, 8)
		for c := 0; c < 8; c++ {
			onesBlock2D[row][c] = ^uint64(0)
		}
	}
	return unravelTall(onesBlock2D, 0)
}

// this is convenient for visualizing steps of the transposition
func genProbeBlock() BitVect {
	probeBlock2D := make([][]uint64, 512)
	for row := range probeBlock2D {
		probeBlock2D[row] = []uint64{0, 1, 2, 3, 4, 5, 6, 7}
	}
	return unravelTall(probeBlock2D, 0)
}

/* TODO - CheckTransposed not working
// test the tester
func TestCheckTransposed(t *testing.T) {

	fmt.Println("orig to tran")
	if !genOrigBlock().CheckTranspose(genTranBlock()) {
		t.Fatalf("Original block is transposed of transposed block but CheckTransposed doesn't identify as such.")
	}

	fmt.Println("tran to orig")
	if !genTranBlock().CheckTranspose(genOrigBlock()) {
		t.Fatalf("Original block is transposed of transposed block but CheckTransposed doesn't identify as such.")
	}
	fmt.Println("ones to ones")
	if !genOnesBlock().CheckTranspose(genOnesBlock()) {
		t.Fatalf("Ones block is transposed of itself but CheckTransposed doesn't identify as such.")
	}
	fmt.Println("orig to orig")
	if genOrigBlock().CheckTranspose(genOrigBlock()) {
		t.Fatalf("Original block is NOT transposed of itself but CheckTransposed doesn't identify as such.")
	}
}
*/

func TestUnReRavelingTall(t *testing.T) {
	trange := []int{512, 512 * 2, 512 * 3, 512 * 4}
	for _, a := range trange {
		matrix := SampleRandomTall(prng, a)
		// TALL m x 8
		// determine number of blocks to split original matrix
		nblks := len(matrix) / 512

		rerav := make([][]uint64, len(matrix))
		for r := range rerav {
			rerav[r] = make([]uint64, len(matrix[0]))
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
	trange := []int{8, 16, 24, 32, 40}
	for _, a := range trange {
		matrix := SampleRandomWide(prng, a)
		// WIDE 512 x n
		// determine number of blocks to split original matrix
		nblks := len(matrix[0]) / 8

		trans := make([][]uint64, len(matrix))
		for r := range trans {
			trans[r] = make([]uint64, len(matrix[0]))
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
		orig := SampleRandomTall(prng, m)
		tr := ConcurrentTranspose(orig, nworkers)
		dtr := ConcurrentTranspose(tr, nworkers)
		// test
		for k := range orig {
			for l := range orig[k] {
				// note the weird aerobics we have to do here because of the residual insignificant rows added
				// due to the encoding of 64 rows in one column element.
				if orig[k][l] != dtr[len(dtr)-len(orig)+k][l] {
					t.Fatal("Doubly-transposed tall (", m, ") matrix did not match with original at row", k, ".")
				}
			}
		}
	}
	// WIDE
	trange = []int{8, 16, 24, 32, 40}
	for _, m := range trange {
		orig := SampleRandomWide(prng, m)
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

/*
func BenchmarkJustTransposeBitVects(b *testing.B) {
	m, _ := unravelMatrix(uintBlock)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, blk := range m {
			blk.transpose()
		}
	}
}
*/
// Test transpose with the added overhead of creating the blocks
// and writing to an output transpose matrix.
func BenchmarkTranspose(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConcurrentTranspose(uintBlock, 1)
	}
}

func BenchmarkConcurrentTranspose(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConcurrentTranspose(uintBlock, nworkers)
	}
}
