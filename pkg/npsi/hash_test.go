package npsi

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
	h, _ := NewHasher(HashSIP, s)
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkMurmur3(b *testing.B) {
	s, _ := makeSalt()
	h, _ := NewHasher(HashMurmur3, s)
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func BenchmarkXXHasher(b *testing.B) {
	s, _ := makeSalt()
	h, _ := NewHasher(HashxxHash, s)
	for i := 0; i < b.N; i++ {
		h.Hash64(xxx)
	}
}

func TestUnknownHasher(t *testing.T) {
	s, _ := makeSalt()
	h, err := NewHasher(666, s)
	if err != ErrUnknownHash {
		t.Fatalf("requested impossible hasher and got %v", h)
	}
}
