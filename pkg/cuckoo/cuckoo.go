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
	// key is log_2(|Y|)
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

// returns the result of h1(x), h2(x), h3(x)
func (c *Cuckoo) hash(item []byte) []uint64 {
	h := make([]uint64, Nhash)

	for i := range h {
		h[i] = uint64(doHash(item, c.seeds[i]))
	}

	return h
}

// need to import hash lib from npsi branch
func dohash(item []byte, seed uint32) {
	return
}

func (c *Cuckoo) Insert(item []byte) {
	return
}

func (c *Cuckoo) tryAdd(item []byte, h *[Nhash]uint64) (added bool) {
	return true
}

func (c *Cuckoo) GetIndexMap() map[[]byte]uint8 {
	return c.z
}

// return m = 1.2 * |Y| + |S|
func (c *Cuckoo) Len() uint64 {
	return c.bucketSize + len(stash)
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
