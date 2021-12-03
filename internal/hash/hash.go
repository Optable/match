package hash

import (
	"encoding/binary"
	"fmt"
	"log"

	metro "github.com/dgryski/go-metro"
	"github.com/optable/match/internal/util"
	"github.com/twmb/murmur3"
)

const (
	SaltLength = 32

	Murmur3 = iota
	Metro
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

// New creates a Hasher of type t
func New(t int, salt []byte) (Hasher, error) {
	switch t {
	case Murmur3:
		return NewMurmur3Hasher(salt)
	case Metro:
		return NewMetroHasher(salt)
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
type metro64 struct {
	seed uint64
}

// NewMetroHasher returns a metro hasher that uses salt as a
// prefix to the bytes being summed
func NewMetroHasher(salt []byte) (metro64, error) {
	if len(salt) != SaltLength {
		return metro64{}, ErrSaltLengthMismatch
	}

	// condense 32 byte salt to a uint64
	seed := make([]byte, 8)
	copy(seed, salt)
	util.Xor(seed, salt[8:16])
	util.Xor(seed, salt[16:24])
	util.Xor(seed, salt[24:])

	return metro64{seed: binary.LittleEndian.Uint64(seed)}, nil
}

func (m metro64) Hash64(p []byte) uint64 {
	return metro.Hash64(p, m.seed)
}
