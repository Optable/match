package cuckoo

import (
	"bytes"
	"crypto/rand"
	"math"
	"testing"
)

func makeSeeds() [Nhash][]byte {
	var seeds [Nhash][]byte

	for i, _ := range seeds {
		seeds[i] = make([]byte, 32)
		if _, err := rand.Read(seeds[i]); err != nil {
			seeds[i] = nil
		}
	}

	return seeds
}

func TestStashSize(t *testing.T) {
	// Table driven test
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
		{uint64(math.Pow(2, 8)), uint64(1.2 * math.Pow(2, 8))},
		{uint64(math.Pow(2, 16)), uint64(1.2 * math.Pow(2, 16))},
	}

	for _, tt := range cuckooTests {
		c := NewCuckoo(tt.size, seeds)
		if c.bucketSize != tt.bSize {
			t.Errorf("cuckoo bucketsize: want: %d, got: %d", tt.bSize, c.bucketSize)
		}

		if !bytes.Equal(c.seeds[0], seeds[0]) {
			t.Errorf("Cuckoo seeds: want: %s, got: %s", string(seeds[0][:]), string(c.seeds[0][:]))
		}
	}
}

func TestInsertiAndGetHashIdx(t *testing.T) {
	seeds := makeSeeds()

	items := [][]byte{
		[]byte("e:0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e"),
		[]byte("e:73244e1b8c426ed93d315034d9332d5326c6b0cd72cc49093c25106f0dd081c5"),
		[]byte("e:e14efb6bb979cd467767d6d90bd9b4f1d901396eabaa90384e747a00d820776d"),
		[]byte("e:402b44cf09b3004c23257d4b9dc39b0a46966865529393f4533048b5e156ad90"),
		[]byte("e:d03ef68830b089a25592cca16fe3eae40a76ddacdd62719c3ff5eb780e4e490f"),
	}

	cuckoo := NewCuckoo(uint64(math.Pow(2, 8)), seeds)
	for _, item := range items {
		err := cuckoo.Insert(item)
		if err != nil {
			t.Errorf("Cuckoo insert failed: %w", err)
		}

		idx, found := cuckoo.GetHashIdx(item)
		if !found {
			t.Errorf("Cuckoo GetHashIdx, item: %s not inserted.", string(item[:]))
		}

		bIdx := cuckoo.bucketIndex(cuckoo.hash(item)[idx])
		if !bytes.Equal(cuckoo.buckets[bIdx], item) {
			t.Errorf("Cuckoo GetHashIdx, hashIdx not correct for item: %s", string(item[:]))
		}
	}
}
