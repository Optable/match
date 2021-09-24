package util

import (
	"fmt"
	"testing"
)

var uintBlock = SampleRandomBlock(prng, 1000)
var randomBlock = From(uintBlock[:512])

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
	return From(origBlock2D)
}

func genTranBlock() BitVect {
	tranBlock2D := make([][]uint64, 512)
	for row := range tranBlock2D {
		tranBlock2D[row] = make([]uint64, 8)
		for c := 0; c < 8; c++ {
			tranBlock2D[row][c] = 0x5555555555555555 // 01010...01
		}
	}
	return From(tranBlock2D)
}

func genOnesBlock() BitVect {
	onesBlock2D := make([][]uint64, 512)
	for row := range onesBlock2D {
		onesBlock2D[row] = make([]uint64, 8)
		for c := 0; c < 8; c++ {
			onesBlock2D[row][c] = ^uint64(0)
		}
	}
	return From(onesBlock2D)
}

func genProbeBlock() BitVect {
	probeBlock2D := make([][]uint64, 512)
	for row := range probeBlock2D {
		probeBlock2D[row] = []uint64{0, 1, 2, 3, 4, 5, 6, 7}
	}
	return From(probeBlock2D)
}

// test the tester
/* TODO: Fix this for the first case.
func TestCheckTransposed(t *testing.T) {
	if !genOrigBlock().CheckTranspose(genTranBlock()) {
		t.Fatalf("Original block is transposed of transposed block but CheckTransposed doesn't identify as such.")
	}
	if !genTranBlock().CheckTranspose(genOrigBlock()) {
		t.Fatalf("Original block is transposed of transposed block but CheckTransposed doesn't identify as such.")
	}
	if !genOnesBlock().CheckTranspose(genOnesBlock()) {
		t.Fatalf("Ones block is transposed of itself but CheckTransposed doesn't identify as such.")
	}
	if genOrigBlock().CheckTranspose(genOrigBlock()) {
		t.Fatalf("Original block is NOT transposed of itself but CheckTransposed doesn't identify as such.")
	}
}
*/

func BenchmarkTranspose(b *testing.B) {
	tr := randomBlock
	fmt.Println("original\n----------")
	tr.PrintBits(128)
	b.StartTimer()
	tr.Transpose()
	b.StopTimer()
	fmt.Println("transposed\n----------")
	tr.PrintBits(128)
	// check if transpose is correct
	tr.Transpose()
	if tr != randomBlock {
		b.Fatalf("Block incorrectly transposed.")
	}
	//if !tr.CheckTranspose(randomBlock) {
	//	b.Fatalf("Block incorrectly transposed.")
	//}
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
