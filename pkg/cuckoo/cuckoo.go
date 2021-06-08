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
func (c *Cuckoo) hash(item []byte) [Nhash]uint64 {
	var hashes [Nhash]uint64

	for i, _ := range hashes {
		hashes[i] = doHash(item, c.seeds[i])
	}

	return hashes
}

// hash item with hash function seeded with seed
func doHash(item []byte, seed []byte) uint64 {
	// error handling?
	h, _ := hash.New(hash.Highway, seed)
	return h.Hash64(item)
}

// wrap hashed val to bucketSize
func (c *Cuckoo) bucketIndex(hash uint64) uint64 {
	return uint64(hash % c.bucketSize)
}

// return the 3 possible bucket indices of an item
func (c *Cuckoo) bucketIndices(item []byte) [Nhash]uint64 {
	var idx [Nhash]uint64
	hashes := c.hash(item)
	for i, h := range hashes {
		idx[i] = c.bucketIndex(h)
	}

	return idx
}

// inserts the item to cuckoo struct
// try to insert the item in free slots first
// otherwise, evict occupied slots, and reinsert evicted item
// as a last resort, push the evicted item onto the stash after
// ReinsertLimit number of reinserts.
// returns an error msg if all failed.
func (c *Cuckoo) Insert(item []byte) error {
	bucketIndices := c.bucketIndices(item)

	// add to free slots
	if c.tryAdd(item, bucketIndices, false, 0) {
		return nil
	}

	// force insert by cuckoo (eviction)
	if homeLessItem, added := c.tryGreedyAdd(item, bucketIndices); added {
		return nil
	} else {
		return fmt.Errorf("Failed to Insert item: %s\n", string(homeLessItem[:]))
	}

}

// find a free slot and insert the item
// if ignore is true, cannot insert into exceptBIdx
func (c *Cuckoo) tryAdd(item []byte, bucketIndices [Nhash]uint64, ignore bool, exceptBIdx uint64) (added bool) {
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

// evict occupied slots, insert item to the evicted slot
// and reinsert the evicted item.
// after ReinsertLimit number of reinserts, push evicted item
// onto the stash
// return false when insert was unsuccessful, with an evicted item
// outside of cuckoo struct.
func (c *Cuckoo) tryGreedyAdd(item []byte, bucketIndices [Nhash]uint64) (homeLessItem []byte, added bool) {
	// we will randomly select an item to evict
	// seed rand
	rand.Seed(time.Now().UnixNano())

	for i := 1; i < ReinsertLimit; i++ {
		// select a random bucket to be evicted
		evictedBIdx := bucketIndices[rand.Intn(Nhash-1)]
		evictedItem := c.buckets[evictedBIdx]
		// insert the item in the evicted slot
		c.buckets[evictedBIdx] = item

		evictedBucketIndices := c.bucketIndices(evictedItem)
		// try to reinsert the evicted items
		// ignore the evictedBIdx since we newly inserted the item there
		if c.tryAdd(evictedItem, evictedBucketIndices, true, evictedBIdx) {
			return nil, true
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
			return nil, true
		}
	}

	// even a stash is not enought to insert the item
	return item, false
}

// given item, find the hash function that gives the bucket idx
// if item is on stash, return StashHidx
func (c *Cuckoo) GetHashIdx(item []byte) (uint8, bool) {
	bucketIndices := c.bucketIndices(item)
	for i, bIdx := range bucketIndices {
		if v, found := c.buckets[bIdx]; found && bytes.Equal(v, item) {
			// the index for hash function is the same as the
			// index for the bucketIndices
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
