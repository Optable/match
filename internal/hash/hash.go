package hash

import (
	"fmt"
	"log"

	"github.com/minio/highwayhash"
	"github.com/twmb/murmur3"
)

const (
	SaltLength = 32

	Murmur3 = iota
	Highway
)

var (
	ErrUnknownHash        = fmt.Errorf("cannot create a hasher of unknown hash type")
	ErrSaltLengthMismatch = fmt.Errorf("provided salt is not %d length", SaltLength)
)

func init() {
	if SaltLength != 32 {
		log.Fatalf("SaltLength has to be fixed to 32 and is set to %d", SaltLength)
	}
}

// Hasher implements different non cryptographic hashing functions
type Hasher interface {
	Hash64([]byte) uint64
}

// New creates a hasher of type t
func New(t int, salt []byte) (Hasher, error) {
	switch t {
	case Murmur3:
		return NewMurmur3Hasher(salt)
	case Highway:
		return NewHighwayHasher(salt)
	default:
		return nil, ErrUnknownHash
	}
}

// Murmur3 implementation of Hasher
type murmur64 struct {
	salt []byte
}

// NewMurmur3Hasher returns a Murmur3 hasher that uses salt as a prefix to the
// bytes being summed
func NewMurmur3Hasher(salt []byte) (murmur64, error) {
	if len(salt) != SaltLength {
		return murmur64{}, ErrSaltLengthMismatch
	}

	return murmur64{salt: salt}, nil
}

func (t murmur64) Hash64(p []byte) uint64 {
	// prepend the salt in m and then Sum
	return murmur3.Sum64(append(t.salt, p...))
}

// Highway Hash implementation of Hasher
type hw struct {
	salt []byte
}

// NewHighwayHasher returns a hw hasher that uses salt as the 4 lanes for the hashing
func NewHighwayHasher(salt []byte) (hw, error) {
	if len(salt) != SaltLength {
		return hw{}, ErrSaltLengthMismatch
	}

	return hw{salt: salt}, nil
}

func (h hw) Hash64(p []byte) uint64 {
	return highwayhash.Sum64(p, h.salt)
}
