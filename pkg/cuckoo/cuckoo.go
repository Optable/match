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

// A Cuckoo represents a 3-way Cuckoo hash table data structure
// that contains buckets and a stash with 3 hash functions
type Cuckoo struct {
	//hashmap (k, v) -> k: h_i(x) (uint64), v: x ([]byte)
	buckets map[uint64][]byte
	// Total bucket count, len(bucket)
	bucketSize uint64
	// 3 hash functions h_0, h_1, h_2
	hashers [Nhash]hash.Hasher
	// array of items
	stash [][]byte
}

// NewCuckoo instantiate the struct Cuckoo with a bucket of size 1.2 * size,
// a stash and 3 seeded hash functions for the 3-way cuckoo hashing.
func NewCuckoo(size uint64, seeds [Nhash][]byte) *Cuckoo {
	bSize := uint64(1.2 * float64(size))
	var hashers [Nhash]hash.Hasher
	for i, s := range seeds {
		hashers[i], _ = hash.New(hash.Highway, s)
	}

	return &Cuckoo{
		buckets:    make(map[uint64][]byte, bSize),
		bucketSize: bSize,
		hashers:    hashers,
		stash:      make([][]byte, findStashSize(size)),
	}
}

// hash returns the result of h0(item), h1(item), h2(item)
func (c *Cuckoo) hash(item []byte) [Nhash]uint64 {
	var hashes [Nhash]uint64

	for i, _ := range hashes {
		hashes[i] = doHash(item, c.hashers[i])
	}

	return hashes
}

// doHash returns the hash of an item given a hash function
func doHash(item []byte, hasher hash.Hasher) uint64 {
	return hasher.Hash64(item)
}

// bucketIndex computes the bucket index
func (c *Cuckoo) bucketIndex(hash uint64) uint64 {
	return uint64(hash % c.bucketSize)
}

// bucketIndices returns the 3 possible bucket indices of an item
func (c *Cuckoo) bucketIndices(item []byte) [Nhash]uint64 {
	var idx [Nhash]uint64
	hashes := c.hash(item)
	for i, h := range hashes {
		idx[i] = c.bucketIndex(h)
	}

	return idx
}

// Insert tries to insert a given item to the bucket
// in available slots, otherwise, it evicts a random occupied slot,
// and reinserts evicted item.
// as a last resort, after ReinsertLim number of reinsetion,
// it pushes the evicted item onto the stash
// returns an error msg if all failed.
func (c *Cuckoo) Insert(item []byte) error {
	bucketIndices := c.bucketIndices(item)

	// if item has already been inserted (duplicated item), exit
	if c.exists(item, bucketIndices) {
		return nil
	}

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

// tryAdd finds a free slot and inserts the item
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

// tryGreedyAdd evicts a random occupied slots, inserts the item to the evicted slot
// and reinserts the evicted item.
// after ReinsertLimit number of reinsertions, it pushes the evicted item onto the stash
// return false and the last evicted item, if reinsertions failed
func (c *Cuckoo) tryGreedyAdd(item []byte, bucketIndices [Nhash]uint64) (homeLessItem []byte, added bool) {
	// seed rand to choose a random occupied slot
	rand.Seed(time.Now().UnixNano())

	for i := 1; i < ReinsertLimit; i++ {
		// select a random slot to be evicted
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

	// even a stash is not enough to insert the item
	return item, false
}

// GetHashIdx finds the index of the hash function {0, 1, 2}
// that gives the bucket index, and a boolean found
// if item is on stash, return StashHidx
// if item is not inserted, return the max value 255, and found=false.
func (c *Cuckoo) GetHashIdx(item []byte) (hIdx uint8, found bool) {
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
		if len(v) > 0 && bytes.Equal(v, item) {
			return uint8(StashHidx), true
		}
	}

	// Not found in bucket nor stash
	return uint8(255), false
}

// exists returns a boolean that signifies an item is already present in cuckoo struct
func (c *Cuckoo) exists(item []byte, bucketIndices [Nhash]uint64) (exists bool) {
	// check bucket
	for _, bIdx := range bucketIndices {
		if v, found := c.buckets[bIdx]; found && bytes.Equal(v, item) {
			return true
		}
	}

	//check stash
	for _, v := range c.stash {
		if len(v) > 0 && bytes.Equal(v, item) {
			return true
		}
	}

	return false
}

// Len returns the total size of the cuckoo struct
// which is equal to bucketSize + stashSize
func (c *Cuckoo) Len() uint64 {
	return c.bucketSize + uint64(len(c.stash))
}

// findStashSize is a helper function that selects the correct stash size
func findStashSize(size uint64) uint8 {
	logSize := uint8(math.Log2(float64(size)))

	switch {
	case logSize > 0 && logSize <= 8:
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
