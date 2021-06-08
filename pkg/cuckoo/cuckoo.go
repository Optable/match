package cuckoo

import (
	"bytes"
	"fmt"
	"github.com/optable/match/internal/hash"
	"math"
	"math/rand"
	"time"
)

const (
	// Nhash is the number of hash function used for cuckoohash
	Nhash         = 3
	ReinsertLimit = 200
	// index returned to signify the item is pushed on to the stash
	// instead of the bucket
	StashHidx = 4
)

var stashSize = map[uint8]uint8{
	// key is log_2(|X|)
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
	// 3 32-bytes salt to instantiate h_1, h_2, h_3
	seeds [Nhash][]byte
	// array of items
	stash [][]byte
}

func NewCuckoo(size uint64, itemByteSize uint8, seeds [Nhash][]byte) *Cuckoo {
	bSize := uint64(1.2 * float64(size))

	return &Cuckoo{
		buckets:    make(map[uint64][]byte, bSize),
		bucketSize: bSize,
		seeds:      seeds,
		stash:      make([][]byte, findStashSize(size)),
	}
}

// returns the result of h1(x), h2(x), h3(x)
func (c *Cuckoo) hash(item []byte) []uint64 {
	h := make([]uint64, Nhash)

	for i, _ := range h {
		h[i] = doHash(item, c.seeds[i])
	}

	return h
}

func doHash(item []byte, seed []byte) uint64 {
	// instantiate hash function seeded with seed
	// error handling?
	h, _ := hash.New(hash.Highway, seed)
	return h.Hash64(item)
}

// wrap hashed val to bucketSize
func (c *Cuckoo) bucketIndex(hash uint64) uint64 {
	return uint64(hash % c.bucketSize)
}

func (c *Cuckoo) Insert(item []byte) {
	var bucketIndices [Nhash]uint64
	hashes := c.hash(item)

	for i, h := range hashes {
		bucketIndices[i] = c.bucketIndex(h)
	}

	// add to free slots
	if c.tryAdd(item, bucketIndices, false, 0) {
		return
	}

	// force insert by cuckoo (eviction)
	if c.tryGreedyAdd(item, bucketIndices) {
		return
	}

	// Failed to insert
	fmt.Printf("Error, cannot insert item: %s\n", string(item[:]))
}

// find a free slot and insert the item
// if ignore is true, cannot insert into exceptBIdx
func (c *Cuckoo) tryAdd(item []byte, bucketIndices [Nhash]uint64, ignore bool, exceptBIdx uint64) (added bool) {
	// Try to insert item except at the exceptBIdx
	for _, bIdx := range bucketIndices {
		if ignore && exceptBIdx == bIdx {
			continue
		}

		if _, occupied := c.buckets[bIdx]; !occupied {
			// this is a free slot
			c.buckets[bIdx] = item
			return true
		}
	}
	return false
}

func (c *Cuckoo) tryGreedyAdd(item []byte, bucketIndices [Nhash]uint64) (added bool) {
	// we will randomly select an item to evict
	// seed rand
	rand.Seed(time.Now().UnixNano())

	for i := 1; i < ReinsertLimit; i++ {
		// select a random bucket to be evicted
		evictedBIdx := bucketIndices[rand.Intn(Nhash-1)]
		evictedItem := c.buckets[evictedBIdx]
		// insert the item in the evicted slot
		c.buckets[evictedBIdx] = item

		// try to reinsert the evicted items
		var evictedBucketIndices [Nhash]uint64
		evictedHashes := c.hash(evictedItem)
		for j, h := range evictedHashes {
			evictedBucketIndices[j] = c.bucketIndex(h)
		}
		// ignore the evictedBIdx since we newly inserted the item there
		if c.tryAdd(evictedItem, evictedBucketIndices, true, evictedBIdx) {
			return true
		}

		// insert evicted item not successful, recurse
		item = evictedItem
		bucketIndices = evictedBucketIndices
	}

	// last resort, push the evicted item onto the stash
	for i, s := range c.stash {
		// empty slot
		if len(s) == 0 {
			c.stash[i] = item
			return true
		}
	}

	// even a stash is not enought to insert the item
	return false
}

// given item, find the hash function that gives the bucket idx
// if item is on stash, return StashHidx
func (c *Cuckoo) GetHashIdx(item []byte) (uint8, bool) {
	hashes := c.hash(item)
	for i, h := range hashes {
		hIdx := c.bucketIndex(h)
		v, found := c.buckets[hIdx]
		if found && bytes.Equal(v, item) {
			return uint8(i), true
		}
	}

	// On stash
	for _, v := range c.stash {
		if bytes.Equal(v, item) {
			return uint8(StashHidx), true
		}
	}

	// Not found in bucket nor stash
	return uint8(0), false
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
