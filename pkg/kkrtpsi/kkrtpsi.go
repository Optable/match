package kkrtpsi

import (
	"github.com/optable/match/internal/hash"
)

// findK returns the number of base OT for OPRF
// these numbers are from the paper: Efficient Batched Oblivious PRF with Applications to Private Set Intersection
// by Vladimir Kolesnikov, Ranjit Kumaresan, Mike Rosulek, and Ni Treu in 2016.
// Reference:	http://dx.doi.org/10.1145/2976749.2978381 (KKRT)
func findK(n int64) int {
	switch {
	// 2^8
	case n > 0 && n <= 256:
		return 424
	// 2^12
	case n > 256 && n <= 4096:
		return 432
	// 2^16
	case n > 4096 && n <= 65536:
		return 440
	// 2^20
	case n > 65536 && n <= 1048576:
		return 448
	case n > 1048576:
		return 448
	default:
		return 128
	}
}

// HashAll reads all inputs from inputs
// and hashes them until inputs closes
func HashAll(h hash.Hasher, input <-chan []byte) <-chan uint64 {
	var hashes = make(chan uint64)

	// just read and hash
	go func() {
		defer close(hashes)
		for in := range input {
			hashes <- h.Hash64(in)
		}
	}()
	return hashes
}
