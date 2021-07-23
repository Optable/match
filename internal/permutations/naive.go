package permutations

import (
	"crypto/rand"
	"math/big"
)

type naive struct {
	p []int64
}

// NewNaive permutation method
func NewNaive(n int64) (naive, error) {
	var p = make([]int64, n)
	var max = big.NewInt(n - 1)
	// Chooses a uniform random int64
	choose := func() (int64, error) {
		if i, err := rand.Int(rand.Reader, max); err == nil {
			return i.Int64(), nil
		} else {
			return 0, err
		}
	}
	// Initialize a trivial permutation
	for i := int64(0); i < n; i++ {
		p[i] = i
	}
	// and then shuffle it by random swaps
	for i := int64(0); i < n; i++ {
		j, err := choose()
		if err != nil {
			return naive{}, err
		}
		if j != i {
			p[j], p[i] = p[i], p[j]
		}
	}

	return naive{p: p}, nil
}

// Shuffle using the naive method
// with n the number to permute/the index of the permutation vector.
func (k naive) Shuffle(n int64) int64 {
	return k.p[n]
}
