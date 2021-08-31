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
	ReinsertLimit = 200
	Factor        = 1.4
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// a value holds the inserted item, and the index of
// the hash function used to compute which bucket index
// the item is inserted in.
type value struct {
	item []byte
	hIdx uint8
}

func (v *value) empty() bool {
	return v == nil
}

// A Cuckoo represents a 3-way Cuckoo hash table data structure
// that contains buckets and a stash with 3 hash functions
type Cuckoo struct {
	//hashmap (k, v) -> k: bucket index, v: value
	buckets []*value
	// Total bucket count, len(bucket)
	bucketSize uint64
	// 3 hash functions h_0, h_1, h_2
	hashers [Nhash]hash.Hasher
}

// NewCuckoo instantiate the struct Cuckoo with a bucket of size 2 * size,
// a stash and 3 seeded hash functions for the 3-way cuckoo hashing.
func NewCuckoo(size uint64, seeds [Nhash][]byte) *Cuckoo {
	bSize := max(1, uint64(Factor*float64(size)))
	//bSize := max(1, uint64(FindBucketSize(size)*float64(size)))
	var hashers [Nhash]hash.Hasher
	for i, s := range seeds {
		hashers[i], _ = hash.New(hash.Highway, s)
	}

	return &Cuckoo{
		buckets:    make([]*value, bSize),
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

// hash returns the result of h0(item), h1(item), h2(item)
func (c *Cuckoo) hash(item []byte) [Nhash]uint64 {
	var hashes [Nhash]uint64

	for i := range hashes {
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
	return hash % c.bucketSize
}

// bucketIndices returns the 3 possible bucket indices of an item
func (c *Cuckoo) BucketIndices(item []byte) (idx [Nhash]uint64) {
	for i, h := range c.hash(item) {
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
		return fmt.Errorf("failed to Insert item: %s", string(homeLessItem[:]))
	}

}

// tryAdd finds a free slot and inserts the item
// if ignore is true, cannot insert into exceptBIdx
func (c *Cuckoo) tryAdd(item []byte, bucketIndices [Nhash]uint64, ignore bool, exceptBIdx uint64) (added bool) {
	for hIdx, bIdx := range bucketIndices {
		if ignore && exceptBIdx == bIdx {
			continue
		}

		if c.buckets[bIdx].empty() {
			// this is a free slot
			c.buckets[bIdx] = &value{item, uint8(hIdx)}
			return true
		}
	}
	return false
}

// tryGreedyAdd evicts a random occupied slots, inserts the item to the evicted slot
// and reinserts the evicted item
// return false and the last evicted item, if reinsertions failed
func (c *Cuckoo) tryGreedyAdd(item []byte, bucketIndices [Nhash]uint64) (homeLessItem []byte, added bool) {
	for i := 1; i < ReinsertLimit; i++ {
		// select a random slot to be evicted
		evictedHIdx := rand.Int31n(Nhash)
		evictedBIdx := bucketIndices[evictedHIdx]
		evictedItem := c.buckets[evictedBIdx]
		// insert the item in the evicted slot
		c.buckets[evictedBIdx] = &value{item, uint8(evictedHIdx)}

		evictedBucketIndices := c.BucketIndices(evictedItem.item)
		// try to reinsert the evicted items
		// ignore the evictedBIdx since we newly inserted the item there
		if c.tryAdd(evictedItem.item, evictedBucketIndices, true, evictedBIdx) {
			return nil, true
		}

		// insert evicted item not successful, recurse
		item = evictedItem.item
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
		if !c.buckets[bIdx].empty() && len(c.buckets[bIdx].item) > 0 && bytes.Equal(c.buckets[bIdx].item, item) {
			// the index for hash function is the same as the
			// index for the bucketIndices
			return c.buckets[bIdx].hIdx, true
		}
	}

	return uint8(255), false
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
	return c.bucketSize //+ uint64(c.StashSize())
}

func (v *value) oprfInput() []byte {
	// no item inserted, return dummy value
	if v.empty() {
		return []byte{255}
	}

	return append(v.item, v.hIdx)
}

// OPRFInput returns the OPRF input for KKRT Receiver
// if the identifier is in the bucket, it appends the hash index
// if the identifier is on stash, it returns just the id
// if the bucket has nothing it in, it returns a dummy value: 255
func (c *Cuckoo) OPRFInput() [][]byte {
	r := make([][]byte, c.Len())
	for i, b := range c.buckets {
		r[i] = b.oprfInput()
	}

	return r
}

func (v *value) GetItem() []byte {
	return v.item
}

func (v *value) GetHashIdx() uint8 {
	return v.hIdx
}

func (c *Cuckoo) Bucket() []*value {
	if c.buckets != nil {
		clone := make([]*value, len(c.buckets))
		copy(clone, c.buckets)
		return clone
	} else {
		return nil
	}
}

func (c *Cuckoo) BucketSize() int {
	return int(c.bucketSize)
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}

	return b
}
