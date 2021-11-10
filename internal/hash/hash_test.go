package hash

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/OneOfOne/xxhash"
	"github.com/minio/highwayhash"
	"github.com/mmcloughlin/meow"
	smurmur "github.com/spaolacci/murmur3"
	tmurmur "github.com/twmb/murmur3"
	"github.com/zeebo/xxh3"
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

func BenchmarkSipHash(b *testing.B) {
	s, _ := makeSalt()
	h, _ := New(SIP, s)
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkTmurmur3(b *testing.B) {
	s, _ := makeSalt()
	h, _ := New(Tmurmur3, s)
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkSmurmur3(b *testing.B) {
	s, _ := makeSalt()
	h, _ := New(Smurmur3, s)
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkXXHasher(b *testing.B) {
	s, _ := makeSalt()
	h, _ := New(XX, s)
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkHighwayHash(b *testing.B) {
	s, _ := makeSalt()
	h, _ := New(Highway, s)
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkXXH3(b *testing.B) {
	s, _ := makeSalt()
	h, _ := New(XXH3, s)
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkHighwayHash16(b *testing.B) {
	s, _ := makeSalt()
	h, _ := highwayhash.New128(s)
	src := make([]byte, 66)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Reset()
		h.Write(src)
		h.Write([]byte{2})
		h.Sum(nil)
	}
}

func BenchmarkMeow16(b *testing.B) {
	src := make([]byte, 66)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		meow.Checksum(2, src)
	}
}

func BenchmarkXXHash16(b *testing.B) {
	src := make([]byte, 66)
	h := xxhash.New64()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Reset()
		h.Write(src)
		h.Write([]byte{2})
		h.Sum(nil)
	}
}

func BenchmarkXXHash316(b *testing.B) {
	src := make([]byte, 66)
	h := xxh3.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Reset()
		h.Write(src)
		h.Write([]byte{2})
		h.Sum128().Bytes()
	}
}

func BenchmarkSpaolacciMurmur316(b *testing.B) {
	src := make([]byte, 66)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hi, lo := smurmur.Sum128WithSeed(src, 2)
		uint128ToBytes(hi, lo)
	}
}

func BenchmarkTwibMurmur316(b *testing.B) {
	src := make([]byte, 66)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hi, lo := tmurmur.SeedSum128(0, 2, src)
		uint128ToBytes(hi, lo)
	}
}

func TestUnknownHasher(t *testing.T) {
	s, _ := makeSalt()
	h, err := New(666, s)
	if err != ErrUnknownHash {
		t.Fatalf("requested impossible hasher and got %v", h)
	}
}

func TestGetSIP(t *testing.T) {
	s, _ := makeSalt()
	h, err := New(SIP, s)
	if err != nil {
		t.Fatalf("got error %v while requesting SIP hash", err)
	}

	if _, ok := h.(siphash64); !ok {
		t.Fatalf("expected type siphash64 and got %T", h)
	}
}

func TestGetTmurmur3(t *testing.T) {
	s, _ := makeSalt()
	h, err := New(Tmurmur3, s)
	if err != nil {
		t.Fatalf("got error %v while requesting murmur3 hash", err)
	}

	if _, ok := h.(tmurmur64); !ok {
		t.Fatalf("expected type murmur64 and got %T", h)
	}
}

func TestGetSmurmur3(t *testing.T) {
	s, _ := makeSalt()
	h, err := New(Smurmur3, s)
	if err != nil {
		t.Fatalf("got error %v while requesting murmur3 hash", err)
	}

	if _, ok := h.(smurmur64); !ok {
		t.Fatalf("expected type murmur64 and got %T", h)
	}
}

func TestGetxxHash(t *testing.T) {
	s, _ := makeSalt()
	h, err := New(XX, s)
	if err != nil {
		t.Fatalf("got error %v while requesting xxHash hash", err)
	}

	if _, ok := h.(xxHash); !ok {
		t.Fatalf("expected type xxHash and got %T", h)
	}
}

func TestGetHighwayHash(t *testing.T) {
	s, _ := makeSalt()
	h, err := New(Highway, s)
	if err != nil {
		t.Fatalf("got error %v while requesting highway hash", err)
	}

	if _, ok := h.(hw); !ok {
		t.Fatalf("expected type hw and got %T", h)
	}
}
