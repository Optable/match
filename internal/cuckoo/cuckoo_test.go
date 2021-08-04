package cuckoo

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

var test_n = uint64(1e6)  // 1Million
var bench_n = uint64(1e7) // 10 Million

var (
	bench_cuckoo *Cuckoo
	bench_data   [][]byte
)

func makeSeeds() [Nhash][]byte {
	var seeds [Nhash][]byte

	for i := range seeds {
		seeds[i] = make([]byte, 32)
		rand.Read(seeds[i])
	}

	return seeds
}

func TestStashSize(t *testing.T) {
	stashSizeTests := []struct {
		n    uint64 //input size
		want uint8  // stash size
	}{
		{uint64(0), uint8(0)},
		{uint64(math.Pow(2, 8) - 1), uint8(12)},
		{uint64(math.Pow(2, 12) - 1), uint8(6)},
		{uint64(math.Pow(2, 16) - 1), uint8(4)},
		{uint64(math.Pow(2, 20) - 1), uint8(3)},
		{uint64(math.Pow(2, 24)), uint8(2)},
		{uint64(math.Pow(2, 25)), uint8(0)},
	}

	for _, tt := range stashSizeTests {
		got := findStashSize(tt.n)
		if got != tt.want {
			t.Errorf("findStashSize(%d): want: %d, got: %d", tt.n, tt.want, got)
		}
	}
}

func TestNewCuckoo(t *testing.T) {
	seeds := makeSeeds()

	cuckooTests := []struct {
		size  uint64
		bSize uint64 //bucketSize
	}{
		{uint64(0), uint64(0)},
		{uint64(math.Pow(2, 4)), uint64(2 * math.Pow(2, 4))},
		{uint64(math.Pow(2, 8)), uint64(2 * math.Pow(2, 8))},
		{uint64(math.Pow(2, 16)), uint64(2 * math.Pow(2, 16))},
	}

	for _, tt := range cuckooTests {
		c := NewCuckoo(tt.size, seeds)
		if c.bucketSize != tt.bSize {
			t.Errorf("cuckoo bucketsize: want: %d, got: %d", tt.bSize, c.bucketSize)
		}
	}
}

func TestInsertAndGetHashIdx(t *testing.T) {
	seeds := makeSeeds()

	cuckoo := NewCuckoo(test_n, seeds)
	data := genBytes(int(test_n))
	errCount := 0

	//test Insert
	for _, item := range data {
		if err := cuckoo.Insert(item); err != nil {
			errCount += 1
		}
	}

	t.Logf("To be inserted: %d, bucketSize: %d, load factor: %f, failure insertion:  %d, stashSize: %d, items on stash: %d\n",
		test_n, cuckoo.bucketSize, cuckoo.LoadFactor(), errCount, len(cuckoo.stash), stashOccupation(cuckoo))

	//test GetHashIdx
	for _, item := range data {
		idx, found := cuckoo.GetHashIdx(item)
		if !found {
			t.Errorf("Cuckoo GetHashIdx, item: %s not inserted.", string(item[:]))
		}

		if idx != StashHidx {
			bIdx := cuckoo.bucketIndex(cuckoo.hash(item)[idx])
			if !bytes.Equal(cuckoo.buckets[bIdx].item, item) {
				t.Errorf("Cuckoo GetHashIdx, hashIdx not correct for item: %s", string(item[:]))
			}
		} else {
			found = false
			for _, v := range cuckoo.stash {
				if bytes.Equal(v.item, item) {
					found = true
				}
			}
			if !found {
				t.Errorf("Cuckoo GetHashIdx, hashIdx is StashHidx but not found in stash for item: %s", string(item[:]))
			}
		}
	}
}

func BenchmarkCuckooInsert(b *testing.B) {
	seeds := makeSeeds()
	bench_cuckoo = NewCuckoo(bench_n, seeds)
	bench_data = genBytes(int(bench_n))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bench_cuckoo.Insert(bench_data[i%int(bench_n)])
	}
}

// Benchmark find hash index
func BenchmarkCuckooGetHashIdx(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bench_cuckoo.GetHashIdx(bench_data[i%int(bench_n)])
	}
}

// Benchmark Exists
func BenchmarkCuckooExists(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bench_cuckoo.Exists(bench_data[i%int(bench_n)])
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

func stashOccupation(c *Cuckoo) int {
	n := 0
	for _, v := range c.stash {
		if len(v.item) > 0 {
			n += 1
		}
	}

	return n
}

func printBucket(c *Cuckoo) {
	for k, v := range c.buckets {
		fmt.Printf("bIdx: %d, item: %s, hIdx:%d\n", k, string(v.item[:]), v.hIdx)
	}
}

func printStash(c *Cuckoo) {
	for _, s := range c.stash {
		fmt.Printf("item: %s, hIdx: %d", string(s.item[:]), s.hIdx)
	}
}
