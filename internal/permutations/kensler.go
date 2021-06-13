package permutations

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
)

/*

Andrew Kensler, a researcher at Pixar, introduced an interesting technique for generating the permutation of an array
in his 2013 paper, Correlated Multi-Jittered Sampling.

reference:             https://graphics.pixar.com/library/MultiJitteredSampling/paper.pdf
further comments from: https://afnan.io/posts/2019-04-05-explaining-the-hashed-permutation/

*/

// kensler shuffler
// l The desired size of the permutation vector
// p The seed of the shuffle
type kensler struct {
	l, p uint32
}

// NewKensler with l the desired size of the permutation vector
func NewKensler(l int64) (kensler, error) {
	if l > math.MaxUint32 {
		return kensler{}, fmt.Errorf("value %d is larger than the maximal value allowable for kensler (%d)", l, math.MaxUint32)
	}
	// make a seed
	var max = big.NewInt(l - 1)
	i, err := rand.Int(rand.Reader, max)
	if err != nil {
		return kensler{}, err
	}

	return kensler{l: uint32(l), p: uint32(i.Int64())}, nil
}

// Shuffle using the kensler algorithm
// with n the number to permute/the index of the permutation vector.
// This is not totally appropriate for our application
// since this only works on uint32 size.
//
// As long as the number of items being matches is not >4b
// its not an issue.
func (k kensler) Shuffle(n int64) int64 {
	var l = k.l
	var p = k.p
	var i = uint32(n)

	var w = l - 1
	w |= w >> 1
	w |= w >> 2
	w |= w >> 4
	w |= w >> 8
	w |= w >> 16

	for {
		i ^= p
		i *= 0xe170893d
		i ^= p >> 16
		i ^= (i & w) >> 4
		i ^= p >> 8
		i *= 0x0929eb3f
		i ^= p >> 23
		i ^= (i & w) >> 1
		i *= 1 | p>>27
		i *= 0x6935fa69
		i ^= (i & w) >> 11
		i *= 0x74dcb303
		i ^= (i & w) >> 2
		i *= 0x9e501cc3
		i ^= (i & w) >> 2
		i *= 0xc860a3df
		i &= w
		i ^= i >> 5

		if i < l {
			break
		}
	}
	return int64((i + p) % l)
}
