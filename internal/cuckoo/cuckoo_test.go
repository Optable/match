package cuckoo

import (
	"bytes"
	"math"
	"math/rand"
	"testing"
	"time"
)

var (
	bench_cuckoo *Cuckoo
	test_n       = uint64(1e6) // 1 Million
	bench_n      = uint64(1e6) // 1 Million
	seeds        = makeSeeds()
	testData     = genBytes(int(test_n))
	benchData    = genBytes(int(bench_n))
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
	cuckoo := NewCuckoo(test_n, seeds)
	errCount := 0

	for _, item := range testData {
		if err := cuckoo.Insert(item); err != nil {
			errCount += 1
		}
	}

	t.Logf("To be inserted: %d, bucketSize: %d, load factor: %f, failure insertion:  %d",
		test_n, cuckoo.bucketSize, cuckoo.LoadFactor(), errCount)

	//test GetHashIdx
	for _, item := range testData {
		hIdx, found := cuckoo.GetHashIdx(item)
		if !found {
			t.Fatalf("Cuckoo GetHashIdx, item: %v not inserted.", item[:])
		}

		bIdx := cuckoo.BucketIndices(item)[hIdx]
		if !bytes.Equal(cuckoo.buckets[bIdx].GetItem(), item) {
			t.Fatalf("Cuckoo GetHashIdx, hashIdx not correct for item: %v, with hIdx: %d, item : %v", item[:], hIdx, cuckoo.buckets[bIdx].item)
		}
	}
}

func BenchmarkCuckooInsert(b *testing.B) {
	seeds := makeSeeds()
	bench_cuckoo = NewCuckoo(bench_n, seeds)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bench_cuckoo.Insert(benchData[i%int(bench_n)])
	}
}

// Benchmark find hash index
func BenchmarkCuckooGetHashIdx(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bench_cuckoo.GetHashIdx(benchData[i%int(bench_n)])
	}
}

// Benchmark Exists
func BenchmarkCuckooExists(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		d := benchData[i%int(bench_n)]
		bench_cuckoo.Exists(d, bench_cuckoo.BucketIndices(d))
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
