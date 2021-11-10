package hash

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"log"

	"github.com/OneOfOne/xxhash"
	"github.com/dchest/siphash"
	"github.com/dgryski/go-highway"
	"github.com/minio/highwayhash"
	smurmur "github.com/spaolacci/murmur3"
	tmurmur "github.com/twmb/murmur3"
	"github.com/zeebo/xxh3"
)

const (
	SaltLength = 32

	SIP = iota
	Smurmur3
	Tmurmur3
	XX
	XXH3
	Highway
	HighwayMinio
	FNV1a
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

// extractSalt a length SaltLength (32 fixed tho) slice of bytes into 4 uint64
//
func extractKeys(salt []byte) (keys []uint64) {
	for i := 0; i < 4; i++ {
		var key = binary.BigEndian.Uint64(salt[i*8 : i*8+8])
		keys = append(keys, key)
	}
	return
}

// uint128ToBytes returns the uint128 as an array of bytes in canonical form (big-endian encoded).
func uint128ToBytes(hi, lo uint64) [16]byte {
	return [16]byte{
		byte(hi >> 0x38), byte(hi >> 0x30), byte(hi >> 0x28), byte(hi >> 0x20),
		byte(hi >> 0x18), byte(hi >> 0x10), byte(hi >> 0x08), byte(hi),
		byte(lo >> 0x38), byte(lo >> 0x30), byte(lo >> 0x28), byte(lo >> 0x20),
		byte(lo >> 0x18), byte(lo >> 0x10), byte(lo >> 0x08), byte(lo),
	}
}

// Hasher implements different non cryptographic
// hashing functions
type Hasher interface {
	Hash64([]byte) uint64
}

// New creates a hasher of type t
func New(t int, salt []byte) (Hasher, error) {
	switch t {
	case SIP:
		return NewSIPHasher(salt)
	case Smurmur3:
		return NewSmurmur3Hasher(salt)
	case Tmurmur3:
		return NewTmurmur3Hasher(salt)
	case XX:
		return NewXXHasher(salt)
	case XXH3:
		return NewXXH3Hasher(salt)
	case Highway:
		return NewHighwayHasher(salt)
	case HighwayMinio:
		return NewHighwayHasherMinio(salt)
	case FNV1a:
		return NewFNV1aHasher(salt)
	default:
		return nil, ErrUnknownHash
	}
}

// FNV1-a implementation of Hasher
type fnv1a struct {
	salt []byte
}

// NewFNV1aHasher returns a FNV1a hasher
// that uses salt as a prefix to the
// bytes being summed
func NewFNV1aHasher(salt []byte) (fnv1a, error) {
	if len(salt) != SaltLength {
		return fnv1a{}, ErrSaltLengthMismatch
	}

	return fnv1a{salt: salt}, nil
}

func (f fnv1a) Hash64(p []byte) uint64 {
	// prepend the salt in m and then Sum
	h := fnv.New64a()
	h.Write(f.salt)
	h.Write(p)
	return h.Sum64()
}

// sipHash implementation of Hasher
type siphash64 struct {
	key0, key1 uint64
}

// NewSIPHasher returns a SIP hasher
// that uses the salt as a key
func NewSIPHasher(salt []byte) (siphash64, error) {
	if len(salt) != SaltLength {
		return siphash64{}, ErrSaltLengthMismatch
	}
	// extract the keys
	keys := extractKeys(salt)
	// xor key0 and key1 into key0, key2 and key3 into key1
	key0 := keys[0] ^ keys[1]
	key1 := keys[2] ^ keys[3]

	return siphash64{key0: key0, key1: key1}, nil
}

func (s siphash64) Hash64(p []byte) uint64 {
	// hash using key0, key1 in s
	return siphash.Hash(s.key0, s.key1, p)
}

// tmurmur3 implementation of Hasher
type tmurmur64 struct {
	salt []byte
}

// NewMurmur3Hasher returns a Murmur3 hasher
// that uses salt as a prefix to the
// bytes being summed
func NewTmurmur3Hasher(salt []byte) (tmurmur64, error) {
	if len(salt) != SaltLength {
		return tmurmur64{}, ErrSaltLengthMismatch
	}

	return tmurmur64{salt: salt}, nil
}

func (t tmurmur64) Hash64(p []byte) uint64 {
	// prepend the salt in m and then Sum
	return tmurmur.Sum64(append(t.salt, p...))
}

// tmurmur3 implementation of Hasher
type smurmur64 struct {
	salt []byte
}

// NewMurmur3Hasher returns a Murmur3 hasher
// that uses salt as a prefix to the
// bytes being summed
func NewSmurmur3Hasher(salt []byte) (smurmur64, error) {
	if len(salt) != SaltLength {
		return smurmur64{}, ErrSaltLengthMismatch
	}

	return smurmur64{salt: salt}, nil
}

func (s smurmur64) Hash64(p []byte) uint64 {
	// prepend the salt in m and then Sum
	return smurmur.Sum64(append(s.salt, p...))
}

// xxHash implementation of Hasher
type xxHash struct {
	salt []byte
}

// NewXXHasher returns a xxHash hasher that uses salt
// as a prefix to the bytes being summed
func NewXXHasher(salt []byte) (xxHash, error) {
	if len(salt) != SaltLength {
		return xxHash{}, ErrSaltLengthMismatch
	}

	return xxHash{salt: salt}, nil
}

func (x xxHash) Hash64(p []byte) uint64 {
	// prepend the salt in x and then Sum
	return xxhash.Checksum64(append(x.salt, p...))
}

// xxHash implementation of Hasher
type xxH3Hash struct {
	salt []byte
}

// NewXXH3Hasher returns a xxHash3 hasher that uses salt
// as a prefix to the bytes being summed
func NewXXH3Hasher(salt []byte) (xxH3Hash, error) {
	if len(salt) != SaltLength {
		return xxH3Hash{}, ErrSaltLengthMismatch
	}

	return xxH3Hash{salt: salt}, nil
}

func (x xxH3Hash) Hash64(p []byte) uint64 {
	// prepend the salt in x and then Sum
	return xxh3.Hash(append(x.salt, p...))
}

// highway hash implementation of Hasher
type hw struct {
	key highway.Lanes
}

// NewHighwayHasher returns a highwayHash hasher that uses salt
// as the 4 lanes for the hashing
func NewHighwayHasher(salt []byte) (hw, error) {
	if len(salt) != SaltLength {
		return hw{}, ErrSaltLengthMismatch
	}

	// extract the keys
	keys := extractKeys(salt)
	// turn into lanes
	var key highway.Lanes
	key[0] = keys[0]
	key[1] = keys[1]
	key[2] = keys[2]
	key[3] = keys[3]

	return hw{key: key}, nil
}

func (h hw) Hash64(p []byte) uint64 {
	// prepend the salt in m and then Sum
	return highway.Hash(h.key, p)
}

type hwMinio struct {
	salt []byte
}

// NewHighwayHasherMinio returns a hwMinio hasher that uses salt
// as the 4 lanes for the hashing
func NewHighwayHasherMinio(salt []byte) (hwMinio, error) {
	if len(salt) != SaltLength {
		return hwMinio{}, ErrSaltLengthMismatch
	}

	return hwMinio{salt: salt}, nil
}

func (h hwMinio) Hash64(p []byte) uint64 {
	return highwayhash.Sum64(p, h.salt)
}
