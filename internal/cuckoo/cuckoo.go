package cuckoo

import (
	"bytes"
	"fmt"
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
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// to check number of collisions for testing purposes
var collision int

// A Cuckoo represents a 3-way Cuckoo hash table data structure
// that contains the items, bucket indices of each item and the 3
// hash functions. The bucket lookup is a lookup table on items which
// tells us which item should be in the bucket at that index. Upon
// construction the items slice has an additional nil value prepended
// so the index of the Cuckoo.items slice is +1 compared to the index
// of the input slice you use.
type Cuckoo struct {
	items        [][]byte
	bucketLookup []uint64
	// Total bucket count, len(bucket)
	bucketSize uint64
	// 3 hash functions h_0, h_1, h_2
	hashers [Nhash]hash.Hasher
}

// NewCuckoo instantiate the struct Cuckoo with a bucket of size 1.4 * size,
// a stash and 3 seeded hash functions for the 3-way cuckoo hashing.
func NewCuckoo(size uint64, seeds [Nhash][]byte) *Cuckoo {
	bSize := max(1, uint64(Factor*float64(size)))
	var hashers [Nhash]hash.Hasher
	for i, s := range seeds {
		hashers[i], _ = hash.New(hash.HighwayMinio, s)
	}

	return &Cuckoo{
		// extra element is "keeper" to which the bucketLookup can be directed
		// when there is no element present in the bucket.
		items:        make([][]byte, size+1),
		bucketLookup: make([]uint64, bSize),
		bucketSize:   bSize,
		hashers:      hashers,
	}
}

// Dummy cuckoo that does not allocate buckets.
func NewDummyCuckoo(size uint64, seeds [Nhash][]byte) *Cuckoo {
	bSize := max(1, uint64(Factor*float64(size)))
	var hashers [Nhash]hash.Hasher
	for i, s := range seeds {
		hashers[i], _ = hash.New(hash.HighwayMinio, s)
	}

	return &Cuckoo{
		bucketSize: bSize,
		hashers:    hashers,
	}
}

// NewTestingCuckoo creates a testing cuckoo that has the same number of
// buckets as identifiers. Each bucket points directly to the respective
// identifier (skipping the nil "keeper" value): bucket 0 -> id 1, bucket
// 1 -> id 2, . . .
func NewTestingCuckoo(buckets [][]byte) *Cuckoo {
	items := make([][]byte, len(buckets)+1)
	copy(items[1:], buckets[:])

	lookup := make([]uint64, len(buckets))
	for i := range lookup {
		lookup[i] = uint64(i + 1)
	}

	return &Cuckoo{
		items:        items,
		bucketSize:   uint64(len(lookup)),
		bucketLookup: lookup,
	}
}

// GetHasher returns the first seeded hash function from a cuckoo struct.
func (c *Cuckoo) GetHasher() hash.Hasher {
	return c.hashers[0]
}

// BucketIndices returns the 3 possible bucket indices of an item
func (c *Cuckoo) BucketIndices(item []byte) (idxs [Nhash]uint64) {
	for i := range idxs {
		idxs[i] = c.hashers[i].Hash64(item) % c.bucketSize
	}

	return idxs
}

// GetBucket returns the index in a given bucket which represents the value in
// the list of identifiers to which it points.
func (c *Cuckoo) GetBucket(bIdx uint64) (uint64, error) {
	if bIdx > c.bucketSize {
		return 0, fmt.Errorf("failed to retrieve item in bucket #%v", bIdx)
	}
	return c.bucketLookup[bIdx], nil
}

// GetItem return the item from it's index in the list
func (c *Cuckoo) GetItem(idx uint64) ([]byte, error) {
	if idx > uint64(len(c.items)-1) {
		return nil, fmt.Errorf("failed to retrieve item #%v", idx)
	}
	if c.items[idx] == nil {
		return nil, fmt.Errorf("failed to retrieve item #%v", idx)
	}
	return c.items[idx], nil
}

// Insert tries to insert a given item (at index, idx) to the bucket
// in available slots, otherwise, it evicts a random occupied slot,
// and reinserts evicted item.
// Returns an error msg if all failed.
func (c *Cuckoo) Insert(input <-chan []byte) error {
	var i uint64 = 1 // skip "keeper" value
	for item := range input {
		err := c.insert(i, item)
		if err != nil {
			return err
		}
		i++
	}
	return nil
}

// insert tries to insert a given item (at index, idx) to the bucket
// in available slots, otherwise, it evicts a random occupied slot,
// and reinserts evicted item.
// Returns an error msg if all failed.
func (c *Cuckoo) insert(idx uint64, item []byte) error {
	c.items[idx] = item
	bucketIndices := c.BucketIndices(item)

	// check if item has already been inserted:
	if _, found := c.Exists(idx); found {
		return nil
	}

	// add to free slots
	if c.tryAdd(idx, bucketIndices, false, 0) {
		return nil
	}

	// force insert by cuckoo (eviction)
	if homelessIdx, added := c.tryGreedyAdd(idx, bucketIndices); added {
		return nil
	} else {
		return fmt.Errorf("failed to Insert item #%v", homelessIdx)
	}
}

// tryAdd finds a free slot and inserts the item (at index, idx)
// if ignore is true, it will not insert into exceptBIdx
func (c *Cuckoo) tryAdd(idx uint64, bucketIndices [Nhash]uint64, ignore bool, exceptBIdx uint64) (added bool) {
	for _, bIdx := range bucketIndices {
		if ignore && exceptBIdx == bIdx {
			continue
		}

		if c.bucketLookup[bIdx] == 0 {
			// this is a free slot
			c.bucketLookup[bIdx] = idx
			return true
		}
	}
	return false
}

// tryGreedyAdd evicts a random occupied slot, inserts the item to the evicted slot
// and reinserts the evicted item. Return false and the last evicted item, if reinsertions
// failed after ReInsertLimit of tries.
func (c *Cuckoo) tryGreedyAdd(idx uint64, bucketIndices [Nhash]uint64) (homeLessItem uint64, added bool) {
	for i := 1; i < ReInsertLimit; i++ {
		// select a random slot to be evicted
		evictedHIdx := rand.Int31n(Nhash)
		evictedBIdx := bucketIndices[evictedHIdx]
		evictedIdx := c.bucketLookup[evictedBIdx]
		// insert the item in the evicted slot
		c.bucketLookup[evictedBIdx] = idx

		evictedBucketIndices := c.BucketIndices(c.items[evictedIdx])
		// try to reinsert the evicted items
		// ignore the evictedBIdx since we newly inserted the item there
		collision++
		if c.tryAdd(evictedIdx, evictedBucketIndices, true, evictedBIdx) {
			return 0, true
		}

		// insert evicted item not successful, recurse
		idx = evictedIdx
		bucketIndices = evictedBucketIndices
	}

	return idx, false
}

// Exists returns the hash index and true if an item is inserted in cuckoo, false otherwise
func (c *Cuckoo) Exists(idx uint64) (hIdx uint8, found bool) {
	item, err := c.GetItem(idx)
	if err != nil {
		return 0, false
	}
	bucketIndices := c.BucketIndices(item)
	for hIdx, bIdx := range bucketIndices {
		if bytes.Equal(c.items[c.bucketLookup[bIdx]], item) {
			return uint8(hIdx), true
		}
	}
	return 0, false
}

// LoadFactor returns the ratio of occupied buckets with the overall bucketSize
func (c *Cuckoo) LoadFactor() (factor float64) {
	occupation := 0
	for _, v := range c.bucketLookup {
		if v != 0 {
			occupation += 1
		}
	}

	return float64(occupation) / float64(c.bucketSize)
}

// Len returns the total size of the cuckoo struct which is equal to bucketSize
func (c *Cuckoo) Len() uint64 {
	return c.bucketSize
}

// IsEmpty returns true if bucket at bidx doesn't contain the index of an identifier
func (c *Cuckoo) IsEmpty(bidx uint64) bool {
	return c.bucketLookup[bidx] != 0
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}

	return b
}
