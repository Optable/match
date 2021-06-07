package hash

import (
	"crypto/rand"
	"fmt"
	"testing"
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

func BenchmarkMurmur3(b *testing.B) {
	s, _ := makeSalt()
	h, _ := New(Murmur3, s)
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
