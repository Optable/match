package npsi

import (
	"encoding/binary"
	"fmt"

	"github.com/dchest/siphash"
	"github.com/spaolacci/murmur3"
)

const (
	SaltLength = 16

	HashSIP = iota
	HashMurmur3
	HashxxHash
)

var (
	ErrUnknownHash        = fmt.Errorf("cannot create a hasher of unknown hash type")
	ErrSaltLengthMismatch = fmt.Errorf("provided salt is not %d length", SaltLength)
)

// Hasher implements different non cryptographic
// hashing functions
type Hasher interface {
	Hash64([]byte) uint64
}

// NewHasher creates a hasher of type t
func NewHasher(t int, salt []byte) (Hasher, error) {
	switch t {
	case HashSIP:
		return NewSIPHasher(salt)
	case HashMurmur3:
		return NewMurmur3Hasher(salt)
	default:
		return nil, ErrUnknownHash
	}
}

// sipHash implementation of Hasher
type siphash64 struct {
	key0, key1 uint64
}

// NewSipHash returns a SIP hasher
// that uses the salt as a key
func NewSIPHasher(salt []byte) (siphash64, error) {
	if len(salt) != SaltLength {
		return siphash64{}, ErrSaltLengthMismatch
	}
	var key0 = binary.BigEndian.Uint64(salt[:SaltLength/2])
	var key1 = binary.BigEndian.Uint64(salt[SaltLength/2 : SaltLength])

	return siphash64{key0: key0, key1: key1}, nil
}

func (s siphash64) Hash64(p []byte) uint64 {
	// hash using key0, key1 in s
	return siphash.Hash(s.key0, s.key1, p)
}

// murmur3 implementation of Hasher
type murmur64 struct {
	salt []byte
}

// NewMurmur3 returns a Murmur3 hasher
// that uses salt as a prefix to the
// bytes being summed
func NewMurmur3Hasher(salt []byte) (murmur64, error) {
	if len(salt) != SaltLength {
		return murmur64{}, ErrSaltLengthMismatch
	}

	return murmur64{salt: salt}, nil
}

func (m murmur64) Hash64(p []byte) uint64 {
	// prepend the salt in m and then Sum
	return murmur3.Sum64(append(m.salt, p...))
}
