package kkrtpsi

import (
	"math"

	"github.com/optable/match/internal/hash"
)

// findK returns the number of base OT for OPRF
func findK(n int64) int {
	logSize := uint8(math.Log2(float64(n)))

	switch {
	case logSize > 0 && logSize <= 8:
		return 424
	case logSize > 8 && logSize <= 12:
		return 432
	case logSize > 12 && logSize <= 16:
		return 440
	case logSize > 16 && logSize <= 20:
		return 448
	case logSize > 20 && logSize <= 24:
		return 448
	default:
		return 128
	}
}

// HashAll reads all identifiers from identifiers
// and hashes them until identifiers closes
func HashAll(h hash.Hasher, input <-chan []byte) <-chan uint64 {
	var hashes = make(chan uint64)

	// just read and hash baby
	go func() {
		defer close(hashes)
		for in := range input {
			hashes <- h.Hash64(in)
		}
	}()
	return hashes
}
