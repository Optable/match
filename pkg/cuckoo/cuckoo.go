package cuckoo

import (
	"math"
	//"github.com/optable/match/pkg/npsi"
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

type Cuckoo struct {
	//hashmap (k, v) -> k: h_i(x) (uint64), v: x ([]byte)
	buckets map[uint64][]byte
	// Total bucket count, len(bucket)
	bucketSize uint64

	seeds [Nhash]uint32

	stash [][]byte

	// map that stores the index of the hash
	// function used to compute the bucket index of the element
	// at index
	// z: (k, v) -> k -> idx, v -> hashidx
	z map[uint64]uint8
}

func NewCuckoo(size uint64, seeds [Nhash]uint32) *Cuckoo {
	bSize := uint64(1.2 * float64(size))

	return &Cuckoo{
		buckets:    make(map[uint64][]byte, bSize),
		bucketSize: bSize,
		seeds:      seeds,
		stash:      make([][]byte, findStashSize(size)),
		z:          make(map[uint64]uint8, size),
	}
}

// returns the result of h1(x), h2(x), h3(x)
func (c *Cuckoo) hash(item []byte) []uint64 {
	h := make([]uint64, Nhash)

	for i := range h {
		h[i] = doHash(item, c.seeds[i])
	}

	return h
}

// need to import hash lib from npsi branch
func doHash(item []byte, seed uint32) uint64 {
	return uint64(0) // to be implemented
}

func (c *Cuckoo) Insert(item []byte) {
	return
}

func (c *Cuckoo) tryAdd(item []byte, h *[Nhash]uint64) (added bool) {
	return true
}

func (c *Cuckoo) GetIndexMap() map[uint64]uint8 {
	return c.z
}

// return m = 1.2 * |Y| + |S|
func (c *Cuckoo) Len() uint64 {
	return c.bucketSize + uint64(len(c.stash))
}

func findStashSize(size uint64) uint8 {
	logSize := uint8(math.Log2(float64(size)))
	//fmt.Printf("size: %d, logsize: %d\n", size, logSize)

	switch {
	case logSize <= 8:
		return stashSize[8]
	case logSize > 8 && logSize <= 12:
		return stashSize[12]
	case logSize > 12 && logSize <= 16:
		return stashSize[16]
	case logSize > 16 && logSize <= 20:
		return stashSize[20]
	case logSize > 20 && logSize <= 24:
		return stashSize[24]
	default:
		return uint8(0)
	}
}
