package hash

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/alecthomas/unsafeslice"
	"github.com/twmb/murmur3"
)

var xxx = []byte("e:0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e")

func makeSalt() ([]byte, error) {
	var s = make([]byte, SaltLength)

	if n, err := rand.Read(s); err != nil {
		return nil, err
	} else if n != SaltLength {
		return nil, fmt.Errorf("requested %d rand bytes and got %d", SaltLength, n)
	} else {
		return s, nil
	}
}

func BenchmarkMurmur3(b *testing.B) {
	s, _ := makeSalt()
	h, _ := NewMurmur3Hasher(s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkMetro(b *testing.B) {
	s, _ := makeSalt()
	h, _ := NewMetroHasher(s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkMurmur316Unsafe(b *testing.B) {
	src := make([]byte, 66)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hi, lo := murmur3.SeedSum128(0, 2, src)
		unsafeslice.ByteSliceFromUint64Slice([]uint64{hi, lo})
	}
}
