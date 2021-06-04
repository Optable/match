package cuckoohash

import (
	"fmt"
)

const (
	// Nhash is the number of hash function used for cuckoohash
	Nhash = 3
)

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

	stash stash //an array of elements
}
