package cuckoo

import (
	"bytes"
	"crypto/rand"
	"math"
	"testing"
	"time"
)

var testN = uint64(1e6) // 1 Million

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

	seeds := makeSeeds()

	for _, tt := range cuckooTests {
		c := NewCuckoo(tt.size, seeds)
		if c.CuckooHasher.bucketSize != tt.bSize {
			t.Errorf("cuckoo bucketsize: want: %d, got: %d", tt.bSize, c.CuckooHasher.bucketSize)
		}
	}
}

func TestInsertAndGetHashIdx(t *testing.T) {
	cuckoo := NewCuckoo(testN, makeSeeds())
	errCount := 0
	testData := genBytes(int(testN))

	insertTime := time.Now()
	for _, item := range testData {
		if err := cuckoo.Insert(item); err != nil {
			errCount += 1
		}
	}

	t.Logf("To be inserted: %d, bucketSize: %d, load factor: %f, failure insertion:  %d, taken %v",
		testN, cuckoo.bucketSize, cuckoo.LoadFactor(), errCount, time.Since(insertTime))

	//test GetHashIdx
	for i, item := range testData {
		bIndices := cuckoo.BucketIndices(item)
		found, hIdx := cuckoo.Exists(item)
		if !found {
			t.Fatalf("Cuckoo GetHashIdx, %dth item: %v not inserted.", i+1, item)
		}

		checkIndex := cuckoo.GetBucket(bIndices[hIdx])
		checkItem, _ := cuckoo.GetItemWithHash(checkIndex)
		if !bytes.Equal(checkItem, item) {
			t.Fatalf("Cuckoo GetHashIdx, hashIdx not correct for item: %v, with hIdx: %d, item : %v", item, hIdx, checkItem)
		}
	}
}

func BenchmarkCuckooInsert(b *testing.B) {
	seeds := makeSeeds()
	benchCuckoo := NewCuckoo(uint64(b.N), seeds)
	benchData := genBytes(int(b.N))
	b.ResetTimer()

	for i := 1; i < b.N; i++ {
		idx := uint64(i % int(b.N))
		if err := benchCuckoo.Insert(benchData[idx]); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark finding hash index and checking existance
func BenchmarkCuckooExists(b *testing.B) {
	seeds := makeSeeds()
	benchCuckoo := NewCuckoo(uint64(b.N), seeds)
	benchData := genBytes(int(b.N))
	b.ResetTimer()

	for i := 1; i < b.N; i++ {
		idx := uint64(i % int(b.N))
		benchCuckoo.Exists(benchData[idx])
	}
}

func genBytes(n int) [][]byte {
	data := make([][]byte, n)
	for i := 0; i < n; i++ {
		data[i] = make([]byte, 64)
		rand.Read(data[i])
	}

	return data
}
