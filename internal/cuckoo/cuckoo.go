package cuckoo

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/optable/match/internal/hash"
)

const (
	// Nhash is the number of hash function used for cuckoohash
	Nhash = 3
	// Maximum number of reinsertons.
	// Each reinsertion kicked off 1 egg (item) and replace it
	// with the item being reinserted, and then reinsert the kicked off egg
	ReInsertLimit = 200
	Factor        = 1.4
	dummyValue    = 255
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// a value holds the inserted item, and the index of
// the hash function used to compute which bucket index
// the item is inserted in.
type value struct {
	item []byte
}

// put inserts x and hashIdx in value by copying x to v.item and appends the hash index to the end
func putValue(hIdx uint8, x []byte) value {
	v := make([]byte, len(x)+1)
	copy(v, x)
	v[len(v)-1] = uint8(hIdx)
	return value{v}
}

// GetItem returns the item in value
func (v value) GetItem() []byte {
	return v.item[:len(v.item)-1]
}

// GetHashIdx returns the hashIndex in value
func (v value) GetHashIdx() uint8 {
	return v.item[len(v.item)-1]
}

func (v value) empty() bool {
	return len(v.item) == 0
}

// A Cuckoo represents a 3-way Cuckoo hash table data structure
// that contains buckets and a stash with 3 hash functions
type Cuckoo struct {
	//hashmap (k, v) -> k: bucket index, v: value
	buckets []value
	// Total bucket count, len(bucket)
	bucketSize uint64
	// 3 hash functions h_0, h_1, h_2
	hashers [Nhash]hash.Hasher
}

// NewCuckoo instantiate the struct Cuckoo with a bucket of size 2 * size,
// a stash and 3 seeded hash functions for the 3-way cuckoo hashing.
func NewCuckoo(size uint64, seeds [Nhash][]byte) *Cuckoo {
	bSize := max(1, uint64(Factor*float64(size)))
	var hashers [Nhash]hash.Hasher
	for i, s := range seeds {
		hashers[i], _ = hash.New(hash.Highway, s)
	}

	return &Cuckoo{
		buckets:    make([]value, bSize),
		bucketSize: bSize,
		hashers:    hashers,
	}
}

// Dummy cuckoo that does not allocate buckets.
func NewDummyCuckoo(size uint64, seeds [Nhash][]byte) *Cuckoo {
	bSize := max(1, uint64(Factor*float64(size)))
	var hashers [Nhash]hash.Hasher
	for i, s := range seeds {
		hashers[i], _ = hash.New(hash.Highway, s)
	}

	return &Cuckoo{
		bucketSize: bSize,
		hashers:    hashers,
	}
}

// libPSI method, added for comparison
func FindBucketSize(size uint64) float64 {
	if size == 0 {
		return 0
	}
	a := 210
	b := math.Log2(float64(size)) - 256

	return (40 - b) / float64(a)
}

// bucketIndices returns the 3 possible bucket indices of an item
func (c *Cuckoo) BucketIndices(item []byte) (idx [Nhash]uint64) {
	for i := range idx {
		idx[i] = c.hashers[i].Hash64(item) % c.bucketSize
	}

	return idx
}

// Insert tries to insert a given item to the bucket
// in available slots, otherwise, it evicts a random occupied slot,
// and reinserts evicted item.
// returns an error msg if all failed.
func (c *Cuckoo) Insert(item []byte) error {
	bucketIndices := c.BucketIndices(item)

	// check if item has already been inserted:
	if found := c.Exists(item, bucketIndices); found {
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
		return fmt.Errorf("failed to Insert item: %v", homeLessItem)
	}
}

// tryAdd finds a free slot and inserts the item
// if ignore is true, it will not insert into exceptBIdx
func (c *Cuckoo) tryAdd(item []byte, bucketIndices [Nhash]uint64, ignore bool, exceptBIdx uint64) (added bool) {
	for hIdx, bIdx := range bucketIndices {
		if ignore && exceptBIdx == bIdx {
			continue
		}

		// this is a free slot
		if c.buckets[bIdx].empty() {
			c.buckets[bIdx] = putValue(uint8(hIdx), item)
			return true
		}
	}
	return false
}

// tryGreedyAdd evicts a random occupied slots, inserts the item to the evicted slot
// and reinserts the evicted item
// return false and the last evicted item, if reinsertions failed after ReInsertLimit of tries.
func (c *Cuckoo) tryGreedyAdd(item []byte, bucketIndices [Nhash]uint64) (homeLessItem []byte, added bool) {
	for i := 1; i < ReInsertLimit; i++ {
		// select a random slot to be evicted
		evictedHIdx := rand.Int31n(Nhash)
		evictedBIdx := bucketIndices[evictedHIdx]
		evictedItem := c.buckets[evictedBIdx].GetItem()
		// insert the item in the evicted slot
		c.buckets[evictedBIdx] = putValue(uint8(evictedHIdx), item)

		evictedBucketIndices := c.BucketIndices(evictedItem)
		// try to reinsert the evicted items
		// ignore the evictedBIdx since we newly inserted the item there
		if c.tryAdd(evictedItem, evictedBucketIndices, true, evictedBIdx) {
			return nil, true
		}

		// insert evicted item not successful, recurse
		item = evictedItem
		bucketIndices = evictedBucketIndices
	}

	return item, false
}

// GetHashIdx finds the index of the hash function {0, 1, 2}
// that gives the bucket index, and a boolean found
// if item is not inserted, return the max value 255, and found is set to false.
func (c *Cuckoo) GetHashIdx(item []byte) (hIdx uint8, found bool) {
	return c.onBucketAtIndex(item, c.BucketIndices(item))
}

// OnBucket returns true if item is inserted in cuckoo hash table.
func (c *Cuckoo) onBucket(item []byte, bucketIndices [Nhash]uint64) (found bool) {
	_, found = c.onBucketAtIndex(item, bucketIndices)
	return found
}

func (c *Cuckoo) onBucketAtIndex(item []byte, bucketIndices [Nhash]uint64) (uint8, bool) {
	for _, bIdx := range bucketIndices {
		if !c.buckets[bIdx].empty() && bytes.Equal(c.buckets[bIdx].GetItem(), item) {
			// the index for hash function is the same as the
			// index for the bucketIndices
			return c.buckets[bIdx].GetHashIdx(), true
		}
	}

	return uint8(dummyValue), false
}

// Exists returns true if an item is inserted in cuckoo, false otherwise
func (c *Cuckoo) Exists(item []byte, bucketIndices [Nhash]uint64) (found bool) {
	return c.onBucket(item, bucketIndices)
}

// LoadFactor returns the ratio of occupied buckets with the overall bucketSize
func (c *Cuckoo) LoadFactor() (factor float64) {
	occupation := 0
	for _, v := range c.buckets {
		if !v.empty() {
			occupation += 1
		}
	}

	return float64(occupation) / float64(c.bucketSize)
}

// Len returns the total size of the cuckoo struct
// which is equal to bucketSize + stashSize
func (c *Cuckoo) Len() uint64 {
	return c.bucketSize
}

// OPRFInput returns the OPRF input for KKRT Receiver
// if the identifier is in the bucket, it appends the hash index
// if the identifier is on stash, it returns just the id
// if the bucket has nothing it in, it returns a dummy value: 255
func (c *Cuckoo) OPRFInput() (input [][]byte) {
	input = make([][]byte, c.bucketSize)
	for i, b := range c.buckets {
		if b.empty() {
			input[i] = []byte{dummyValue}
		} else {
			input[i] = b.item
		}
	}

	return input
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}

	return b
}
