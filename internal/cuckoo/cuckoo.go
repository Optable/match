package cuckoo

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/optable/match/internal/hash"
)

const (
	// Nhash is the number of hash function used for cuckoo hash
	Nhash = 3
	// ReInsertLimit is the maximum number of reinsertions.
	// Each reinsertion kicks off 1 egg (item) and replace it
	// with the item being reinserted, and then reinserts the
	// kicked off egg
	ReInsertLimit = 200
	// Factor is the multiplicative factor of items to be
	// inserted which represents the capacity overhead of
	// the hash table to reduce risk of failure on insertion.
	Factor = 1.4
)

// CuckooHasher is the building block of a Cuckoo hash table. It only holds
// the bucket size and the hashers.
type CuckooHasher struct {
	// Total bucket count, len(bucket)
	bucketSize uint64
	// 3 hash functions h_0, h_1, h_2
	hashers [Nhash]hash.Hasher
}

// NewCuckooHasher instantiates a CuckooHasher struct.
func NewCuckooHasher(size uint64, seeds [Nhash][]byte) *CuckooHasher {
	// get randombyte from crypto/rand
	var rb [8]byte
	if _, err := crand.Read(rb[:]); err != nil {
		panic(err)
	}

	// WARNING: math/rand is not concurrency-safe
	// replace with crypto/rand if that's what you want

	// seed math/rand with crypto/rand
	rand.Seed(int64(binary.LittleEndian.Uint64(rb[:])))

	bSize := max(1, uint64(Factor*float64(size)))
	var hashers [Nhash]hash.Hasher
	var err error
	for i, s := range seeds {
		if hashers[i], err = hash.NewMetroHasher(s); err != nil {
			panic(err)
		}
	}

	return &CuckooHasher{
		bucketSize: bSize,
		hashers:    hashers,
	}
}

// GetHasher returns the first seeded hash function from a CuckooHasher struct.
func (h *CuckooHasher) GetHasher() hash.Hasher {
	return h.hashers[0]
}

// BucketIndices returns the 3 possible bucket indices of an item
func (h *CuckooHasher) BucketIndices(item []byte) (idxs [Nhash]uint64) {
	for i := range idxs {
		idxs[i] = h.hashers[i].Hash64(item) % h.bucketSize
	}

	return idxs
}

// Cuckoo represents a 3-way Cuckoo hash table data structure
// that contains the items, bucket indices of each item and the 3
// hash functions. The bucket lookup is a lookup table on items which
// tells us which item should be in the bucket at that index. Upon
// construction the items slice has an additional nil value prepended
// so the index of the Cuckoo.items slice is +1 compared to the index
// of the input slice you use. The number of inserted items is also
// tracked.
type Cuckoo struct {
	items        [][]byte
	inserted     uint64
	hashIndices  []byte
	bucketLookup []uint64
	*CuckooHasher
}

// NewCuckoo instantiates a Cuckoo struct with a bucket of size Factor * size,
// and a CuckooHasher for the 3-way cuckoo hashing.
func NewCuckoo(size uint64, seeds [Nhash][]byte) *Cuckoo {
	cuckooHasher := NewCuckooHasher(size, seeds)

	return &Cuckoo{
		// extra element is "keeper" to which the bucketLookup can be directed
		// when there is no element present in the bucket.
		make([][]byte, size+1),
		0,
		make([]byte, size+1),
		make([]uint64, cuckooHasher.bucketSize),
		cuckooHasher,
	}
}

// GetBucket returns the index in a given bucket which represents the value in
// the list of identifiers to which it points.
func (c *Cuckoo) GetBucket(bIdx uint64) uint64 {
	if bIdx > c.bucketSize {
		panic(fmt.Errorf("failed to retrieve item in bucket #%v", bIdx))
	}
	return c.bucketLookup[bIdx]
}

// GetItemWithHash returns the item at a given index along with its
// hash index. Panic if the index is greater than the number of items.
func (c *Cuckoo) GetItemWithHash(idx uint64) (item []byte, hIdx uint8) {
	if idx > uint64(len(c.items)-1) {
		panic(fmt.Errorf("index greater than number of items"))
	}

	return c.items[idx], c.hashIndices[idx]
}

// Exists returns true if an item is inserted in cuckoo, false otherwise
func (c *Cuckoo) Exists(item []byte) (bool, byte) {
	bucketIndices := c.BucketIndices(item)

	for hIdx, bIdx := range bucketIndices {
		if bytes.Equal(c.items[c.bucketLookup[bIdx]], item) {
			return true, byte(hIdx)
		}
	}
	return false, 0
}

// Insert tries to insert a given item at the next index to the bucket
// in available slots, otherwise, it evicts a random occupied slot,
// and reinserts evicted item.
// Returns an error msg if all failed.
func (c *Cuckoo) Insert(item []byte) error {
	if int(c.inserted) == len(c.items) {
		return fmt.Errorf("%v of %v items have already been inserted into the cuckoo hash table. Cannot insert again", c.inserted, len(c.items))
	}
	c.items[c.inserted+1] = item
	bucketIndices := c.BucketIndices(item)

	// check if item has already been inserted
	if found, _ := c.Exists(item); found {
		return nil
	}

	// add to free slots
	if c.tryAdd(c.inserted+1, bucketIndices, false, 0) {
		c.inserted++
		return nil
	}

	// force insert by cuckoo (eviction)
	homelessIdx, added := c.tryGreedyAdd(c.inserted+1, bucketIndices)
	if added {
		c.inserted++
		return nil
	}

	return fmt.Errorf("failed to Insert item %v, results in homeless item #%v", item, homelessIdx)
}

// tryAdd finds a free slot and inserts the item (at index, idx)
// if ignore is true, it will not insert into exceptBIdx
func (c *Cuckoo) tryAdd(idx uint64, bucketIndices [Nhash]uint64, ignore bool, exceptBIdx uint64) (added bool) {
	for hIdx, bIdx := range bucketIndices {
		if ignore && exceptBIdx == bIdx {
			continue
		}

		if c.isEmpty(bIdx) {
			// this is a free slot
			c.bucketLookup[bIdx] = idx
			c.hashIndices[idx] = uint8(hIdx)
			return true
		}
	}
	return false
}

// tryGreedyAdd evicts a random occupied slot, inserts the item to the evicted slot
// and reinserts the evicted item. If reinsertions fail after ReInsertLimit tries
// return false and the last evicted item.
func (c *Cuckoo) tryGreedyAdd(idx uint64, bucketIndices [Nhash]uint64) (homeLessItem uint64, added bool) {
	for i := 1; i < ReInsertLimit; i++ {
		// select a random slot to be evicted
		// replace me with crypto/rand for concurrent safety
		evictedHIdx := rand.Intn(Nhash)
		evictedBIdx := bucketIndices[evictedHIdx]
		evictedIdx := c.bucketLookup[evictedBIdx]
		// insert the item in the evicted slot
		c.bucketLookup[evictedBIdx] = idx
		c.hashIndices[idx] = byte(evictedHIdx)

		evictedBucketIndices := c.BucketIndices(c.items[evictedIdx])
		// try to reinsert the evicted items
		// ignore the evictedBIdx since we just inserted there
		if c.tryAdd(evictedIdx, evictedBucketIndices, true, evictedBIdx) {
			return 0, true
		}

		// insertion of evicted item unsuccessful, recurse
		idx = evictedIdx
		bucketIndices = evictedBucketIndices
	}

	return idx, false
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

// Len returns the total size of the cuckoo struct which is equal
// to bucketSize
func (c *Cuckoo) Len() uint64 {
	return c.bucketSize
}

// isEmpty returns true if bucket at bidx does not contain the index
//  of an identifier
func (c *Cuckoo) isEmpty(bidx uint64) bool {
	return c.bucketLookup[bidx] == 0
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}

	return b
}
