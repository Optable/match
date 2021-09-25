package util

import (
	"testing"
)

var uintBlock = SampleRandomTall(prng, 1000)
var randomBlock = Unravel(uintBlock, 0, 0)

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
	return Unravel(origBlock2D, 0, 0)
}

func genTranBlock() BitVect {
	tranBlock2D := make([][]uint64, 512)
	for row := range tranBlock2D {
		tranBlock2D[row] = make([]uint64, 8)
		for c := 0; c < 8; c++ {
			tranBlock2D[row][c] = 0x5555555555555555 // 01010...01
		}
	}
	return Unravel(tranBlock2D, 0, 0)
}

func genOnesBlock() BitVect {
	onesBlock2D := make([][]uint64, 512)
	for row := range onesBlock2D {
		onesBlock2D[row] = make([]uint64, 8)
		for c := 0; c < 8; c++ {
			onesBlock2D[row][c] = ^uint64(0)
		}
	}
	return Unravel(onesBlock2D, 0, 0)
}

// this is convenient for visualizing steps of the transposition
func genProbeBlock() BitVect {
	probeBlock2D := make([][]uint64, 512)
	for row := range probeBlock2D {
		probeBlock2D[row] = []uint64{0, 1, 2, 3, 4, 5, 6, 7}
	}
	return Unravel(probeBlock2D, 0, 0)
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

func TestUnReRaveling(t *testing.T) {
	trange := []int{200, 511, 512, 513, 710, 5120, 5320}
	//trange := []int{513}
	// TALL m x 512
	for _, r := range trange {
		orig := SampleRandomTall(prng, r)
		m, mp := UnravelMatrix(orig)
		if mp != 512-(r%512) && mp != 0 {
			t.Fatal("Unraveling a tall (", r, ") matrix did not result in", 512-(r%512), "or 0 rows of padding.")
		}
		var padded int
		if mp > 0 {
			padded = 1
		}
		if len(m) != (r/512 + padded) {
			t.Fatal("Unraveling a tall (", r, ") matrix did not result in", r/512+padded, "blocks of 512x512.")
		}

		// now reconstruct
		rerav := make([][]uint64, r)
		for row := range rerav {
			rerav[row] = make([]uint64, 8)
		}

		// padded block first
		if padded == 1 {
			m[0].Ravel(rerav, mp, 0)
		} else {
			m[0].Ravel(rerav, 0, 0)
		}

		// rest
		for blk := 0; blk < len(m)-1; blk++ {
			m[blk+1].Ravel(rerav, 0, (512-mp)+(blk*512))
		}

		for k := range rerav {
			for l := range rerav[k] {
				//fmt.Println("k", k, "l", l)
				if rerav[k][l] != orig[k][l] {
					t.Fatal("Unraveled and reraveled tall (", r, ") matrix did not match with original at row", k, ".")
				}
			}
		}
	}
	trange = []int{8, 9, 14, 80, 83}
	//trange = []int{9}
	// WIDE 512 x n
	for _, c := range trange {
		orig := SampleRandomWide(prng, c)
		m, mp := UnravelMatrix(orig)
		if mp != 8-(c%8) && mp != 0 {
			t.Fatal("Unraveling a wide (", c, ") matrix did not result in", 8-(c%8), "or", "0 columns of padding.")
		}
		var padded int
		if mp > 0 {
			padded = 1
		}
		if len(m) != (c/8 + padded) {
			t.Fatal("Unraveling a wide (", c, ") matrix did not result in", c/8+padded, "blocks of 512x512.")
		}

		// now reconstruct
		rerav := make([][]uint64, 512)
		for row := range rerav {
			rerav[row] = make([]uint64, c)
		}

		// padded block first
		if padded == 1 {
			m[0].Ravel(rerav, mp, 0)
		} else {
			m[0].Ravel(rerav, 0, 0)
		}

		// rest
		for blk := 0; blk < len(m)-1; blk++ {
			m[blk+1].Ravel(rerav, 0, (8-mp)+(blk*8))
		}

		for k := range rerav {
			for l := range rerav[k] {
				//fmt.Println("k", k, "l", l)
				if rerav[k][l] != orig[k][l] {
					t.Fatal("Unraveled and reraveled wide (", c, ") matrix did not match with original at row", k, ".")
				}
			}
		}
	}
}

func TestTranspose(t *testing.T) {
	tr := randomBlock

	tr.Transpose()
	tr.Transpose()
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

func BenchmarkTranspose(b *testing.B) {
	tr := randomBlock
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Transpose()
	}
}

/*
func BenchmarkXorBytes(b *testing.B) {
	a := make([]byte, 10000)
	SampleBitSlice(prng, a)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		XorBytes(a, a)
	}
}
*/
