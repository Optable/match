package hash

import (
	"encoding/binary"
	"fmt"
	"log"

	metro "github.com/dgryski/go-metro"
	"github.com/optable/match/internal/util"
	"github.com/twmb/murmur3"
)

// SaltLength is the number of bytes which should be used as salt in
// hashing functions
const SaltLength = 32

// ErrSaltLengthMismath is used when the provided salt does not match
// the expected SaltLength
var ErrSaltLengthMismatch = fmt.Errorf("provided salt is not %d length", SaltLength)

func init() {
	if SaltLength != 32 {
		log.Fatalf("SaltLength has to be fixed to 32 and is set to %d", SaltLength)
	}
}

// Hasher implements different non cryptographic hashing functions
type Hasher interface {
	Hash64([]byte) uint64
}

// Murmur3 implementation of Hasher
type murmur64 struct {
	salt []byte
}

// NewMurmur3Hasher returns a Murmur3 hasher that uses salt as a prefix to the
// bytes being summed
func NewMurmur3Hasher(salt []byte) (Hasher, error) {
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
func NewMetroHasher(salt []byte) (Hasher, error) {
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
