package cuckoohash

import (
	"fmt"
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
	values [][]byte
}

// a hash table with key = h_i(x), value = x
type bucket struct {
	key   uint64
	value []byte
}

type CuckooTable struct {
	//hashmap (k, v) -> k: uint64, v: []byte
	buckets []bucket
	// Total bucket count, len(bucket)
	bucketCount uint64

	seeds [Nhash]uint32

	stash stash

	// map that stores the index of the hash
	// function used to compute the bucket index of the element
	z map[[]byte]uint8
}
