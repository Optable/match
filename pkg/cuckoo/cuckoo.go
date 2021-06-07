package cuckoo

import (
	"math"
)

const (
	// Nhash is the number of hash function used for cuckoohash
	Nhash         = 3
	ReinsertLimit = 200
)

var stashSize = map[uint8]uint8{
	// key is N in power in base 2
	// value is # of elements in stash
	// values taken from Phasing: PSI using permutation-based hashing
	8:  12,
	12: 6,
	16: 4,
	20: 3,
	24: 2,
}

// an array of input elements
type stash struct {
	value []byte
}

// a hash table with key = h_i(x), value = x
type bucket struct {
	key   uint64
	value []byte
}

type Cuckoo struct {
	//hashmap (k, v) -> k: uint64, v: []byte
	buckets []bucket
	// Total bucket count, len(bucket)
	bucketSize uint64

	seeds [Nhash]uint32

	stash []stash

	// map that stores the index of the hash
	// function used to compute the bucket index of the element
	z map[[]byte]uint8
}

func NewCuckoo(size uint64, seeds []uint32) *Cuckoo {
	bsize := uint64(1.2 * size)

	return &Cuckoo{
		buckets:    make(bucket, bsize),
		bucketSize: bsize,
		seeds:      seeds,
		stash:      make(stash, findStashSize(size)),
		z:          make(map[[]byte]uint8, size),
	}
}

func findStashSize(size uint64) uint8 {
	logSize := uint8(math.Log2(size))

	switch logSize {
	case logSize <= 8:
		return stashSize[8]
	case logSize > 8 && logSize <= 12:
		return stashSize[12]
	case logSize > 12 && logSize <= 16:
		return stashSize[16]
	case logSize > 16 && logSize <= 20:
		return stashSize[20]
	case logSize > 20 && logSize <= 24:
		return stashSize[2]
	default:
		return uint8(0)
	}
}
