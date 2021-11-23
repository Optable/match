package hash

import (
	"fmt"
	"log"

	"github.com/dgryski/go-metro"
	"github.com/minio/highwayhash"
	"github.com/shivakar/metrohash"
	"github.com/twmb/murmur3"
)

const (
	SaltLength = 32

	Murmur3 = iota
	Highway
	Metro
	ShivMetro
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
	case Metro:
		return NewMetroHasher(salt)
	case ShivMetro:
		return NewShivMetroHasher(salt)
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

// Metro Hash implementation of Hasher
type metro64 struct {
	salt []byte
}

// NewMetroHasher returns a metro hasher that uses salt as a prefix to the
// bytes being summed
func NewMetroHasher(salt []byte) (metro64, error) {
	if len(salt) != SaltLength {
		return metro64{}, ErrSaltLengthMismatch
	}

	return metro64{salt: salt}, nil
}

func (m metro64) Hash64(p []byte) uint64 {
	return metro.Hash64(append(m.salt, p...), 0)
}

// Metro Hash implementation of Hasher
type shivMetro64 struct {
	salt []byte
}

// NewShivMetroHasher returns a shivMetro64 hasher that uses salt as a
// prefix to the bytes being summed
func NewShivMetroHasher(salt []byte) (shivMetro64, error) {
	if len(salt) != SaltLength {
		return shivMetro64{}, ErrSaltLengthMismatch
	}

	return shivMetro64{salt: salt}, nil
}

func (m shivMetro64) Hash64(p []byte) uint64 {
	h := metrohash.NewMetroHash64()
	h.Write(m.salt)
	h.Write(p)
	return h.Sum64()
}
