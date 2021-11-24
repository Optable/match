package hash

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/optable/match/internal/util"
	"github.com/shivakar/metrohash"
	"github.com/twmb/murmur3"
)

const (
	SaltLength = 32

	Murmur3 = iota
	Metro
	MetroCached
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
	case Metro:
		return NewMetroHasher(salt)
	case MetroCached:
		return NewMetroCachedHasher(salt)
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

// Metro Hash implementation of Hasher
type metro struct {
	salt []byte
}

// NewMetroHasher returns a metro hasher that uses salt as a
// prefix to the bytes being summed
func NewMetroHasher(salt []byte) (metro, error) {
	if len(salt) != SaltLength {
		return metro{}, ErrSaltLengthMismatch
	}

	return metro{salt: salt}, nil
}

func (m metro) Hash64(p []byte) uint64 {
	h := metrohash.NewMetroHash64()
	h.Write(m.salt)
	h.Write(p)
	return h.Sum64()
}

// Metro Hash implementation of Hasher
type metroCached struct {
	hasher *metrohash.MetroHash64
}

// NewMetroCachedHasher returns a metro hasher that uses salt internally
func NewMetroCachedHasher(salt []byte) (metroCached, error) {
	if len(salt) != SaltLength {
		return metroCached{}, ErrSaltLengthMismatch
	}

	// condense 32 byte salt to a uint64
	seed := make([]byte, 8)
	copy(seed, salt)
	util.Xor(seed, salt[8:16])
	util.Xor(seed, salt[16:24])
	util.Xor(seed, salt[24:])

	return metroCached{hasher: metrohash.NewSeedMetroHash64(binary.LittleEndian.Uint64(seed))}, nil
}

func (m metroCached) Hash64(p []byte) uint64 {
	m.hasher.Reset()
	m.hasher.Write(p)
	return m.hasher.Sum64()
}
