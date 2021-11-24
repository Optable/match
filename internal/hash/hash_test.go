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

func TestUnknownHasher(t *testing.T) {
	s, _ := makeSalt()
	h, err := New(666, s)
	if err != ErrUnknownHash {
		t.Fatalf("requested impossible hasher and got %v", h)
	}
}

func TestGetMurmur3(t *testing.T) {
	s, _ := makeSalt()
	h, err := New(Murmur3, s)
	if err != nil {
		t.Fatalf("got error %v while requesting murmur3 hash", err)
	}

	if _, ok := h.(murmur64); !ok {
		t.Fatalf("expected type murmur64 and got %T", h)
	}
}

func TestGetMetro(t *testing.T) {
	s, _ := makeSalt()
	h, err := New(Metro, s)
	if err != nil {
		t.Fatalf("got error %v while requesting metro hash", err)
	}

	if _, ok := h.(metro64); !ok {
		t.Fatalf("expected type metro64 and got %T", h)
	}
}

func BenchmarkMurmur3(b *testing.B) {
	s, _ := makeSalt()
	h, _ := New(Murmur3, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkMetro(b *testing.B) {
	s, _ := makeSalt()
	h, _ := New(Metro, s)
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
