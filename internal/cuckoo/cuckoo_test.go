package cuckoo

import (
	"bytes"
	"math"
	"math/rand"
	"testing"
	"time"
)

var (
	benchCuckoo *Cuckoo
	testN       = uint64(1e6) // 1 Million
	benchN      = uint64(1e6) // 1 Million
	seeds       = makeSeeds()
	testData    = genBytes(int(testN))
	benchData   = genBytes(int(benchN))
)

func makeSeeds() [Nhash][]byte {
	var seeds [Nhash][]byte

	for i := range seeds {
		seeds[i] = make([]byte, 32)
		rand.Read(seeds[i])
	}

	return seeds
}

func TestNewCuckoo(t *testing.T) {
	cuckooTests := []struct {
		size  uint64
		bSize uint64 //bucketSize
	}{
		{uint64(0), uint64(1)},
		{uint64(math.Pow(2, 4)), uint64(Factor * math.Pow(2, 4))},
		{uint64(math.Pow(2, 8)), uint64(Factor * math.Pow(2, 8))},
		{uint64(math.Pow(2, 16)), uint64(Factor * math.Pow(2, 16))},
	}

	for _, tt := range cuckooTests {
		c := NewCuckoo(tt.size, seeds)
		if c.bucketSize != tt.bSize {
			t.Errorf("cuckoo bucketsize: want: %d, got: %d", tt.bSize, c.bucketSize)
		}
	}
}

func TestInsertAndGetHashIdx(t *testing.T) {
	cuckoo := NewCuckoo(testN, seeds)
	errCount := 0

	insertTime := time.Now()
	for idx, item := range testData {
		if err := cuckoo.insert(uint64(idx+1), item); err != nil {
			errCount += 1
		}
	}

	t.Logf("To be inserted: %d, bucketSize: %d, load factor: %f, failure insertion:  %d, collisions: %d, taken %v",
		testN, cuckoo.bucketSize, cuckoo.LoadFactor(), errCount, collision, time.Since(insertTime))

	//test GetHashIdx
	for i, item := range testData {
		bIndices := cuckoo.BucketIndices(item)
		hIdx, found := cuckoo.Exists(uint64(i + 1))
		if !found {
			t.Fatalf("Cuckoo GetHashIdx, %dth item: %v not inserted.", i+1, item)
		}

		checkIndex, _ := cuckoo.GetBucket(bIndices[hIdx])
		checkItem, err := cuckoo.GetItem(checkIndex)
		if !bytes.Equal(checkItem, item) || err != nil {
			t.Fatalf("Cuckoo GetHashIdx, hashIdx not correct for item: %v, with hIdx: %d, item : %v", item, hIdx, checkItem)
		}
	}
}

func BenchmarkCuckooInsert(b *testing.B) {
	seeds := makeSeeds()
	benchCuckoo = NewCuckoo(benchN, seeds)
	b.ResetTimer()

	for i := 1; i < b.N; i++ {
		idx := uint64(i % int(benchN))
		benchCuckoo.insert(idx, benchData[idx])
	}
}

// Benchmark finding hash index and checking existance
func BenchmarkCuckooExists(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchCuckoo.Exists(uint64(i % int(benchN)))
	}
}

func genBytes(n int) [][]byte {
	rand.Seed(time.Now().UnixNano())
	data := make([][]byte, n)
	for i := 0; i < n; i++ {
		data[i] = make([]byte, 64)
		rand.Read(data[i])
	}

	return data
}
