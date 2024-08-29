package pair

import (
	"crypto"
	"errors"

	"github.com/gtank/ristretto255"
)

type PAIRMode uint8

const (
	// PAIRSHA256Ristretto25519 is PAIR with SHA256 as hash function and Ristretto25519 as curve.
	PAIRSHA256Ristretto25519 PAIRMode = 0x01
	// PAIRSHA512Ristretto25519 is PAIR with SHA512 as hash function and Ristretto25519 as curve.
	PAIRSHA512Ristretto25519 PAIRMode = 0x02
)

const (
	sha256SaltSize         = 32
	sha512SaltSize         = 64
	ristretto255ScalarSize = 32
)

var (
	ErrInvalidPAIRMode   = errors.New("invalid PAIR mode")
	ErrInvalidSaltSize   = errors.New("invalid hash salt size")
	ErrInvalidPrivateKey = errors.New("invalid private key")
)

// PrivateKey represents a ristrtto25519 private key
// and a random salt for the internal hash function.
type PrivateKey struct {
	// h is the hash function used to hash the data
	h crypto.Hash

	// salt for h
	salt []byte

	// private key
	scalar *ristretto255.Scalar
}

func (p PAIRMode) New(salt []byte, scalar []byte) (*PrivateKey, error) {
	pk := new(PrivateKey)

	switch p {
	case PAIRSHA256Ristretto25519:
		pk.h = crypto.SHA256
		if len(salt) != sha256SaltSize {
			return nil, ErrInvalidSaltSize
		}
		pk.salt = salt
	case PAIRSHA512Ristretto25519:
		pk.h = crypto.SHA512
		if len(salt) != sha512SaltSize {
			return nil, ErrInvalidSaltSize
		}
		pk.salt = salt
	default:
		return nil, ErrInvalidPAIRMode
	}

	if len(scalar) != ristretto255ScalarSize {
		return nil, ErrInvalidPrivateKey
	}

	pk.scalar = ristretto255.NewScalar()
	if err := pk.scalar.UnmarshalText(scalar); err != nil {
		return nil, ErrInvalidPrivateKey
	}

	return pk, nil
}
