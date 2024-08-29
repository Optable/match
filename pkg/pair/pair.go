package pair

import (
	"crypto"
	"crypto/sha512"
	"errors"
	"hash"

	"github.com/gtank/ristretto255"
)

type PAIRMode uint8

const (
	// PAIRSHA256Ristretto25519 is PAIR with SHA256 as hash function and Ristretto255 as the group.
	PAIRSHA256Ristretto25519 PAIRMode = 0x01
)

const (
	sha256SaltSize = 32
)

var (
	ErrInvalidPAIRMode = errors.New("invalid PAIR mode")
	ErrInvalidSaltSize = errors.New("invalid hash salt size")
)

// PrivateKey represents a PAIR private key.
type PrivateKey struct {
	// h is the hash function used to hash the data
	h hash.Hash

	// salt for h
	salt []byte

	// private key
	scalar *ristretto255.Scalar
}

// New instantiates a new private key with the given salt and scalar.
// It expects the scalar to be base64 encoded.
func (p PAIRMode) New(salt []byte, scalar []byte) (*PrivateKey, error) {
	pk := new(PrivateKey)

	switch p {
	case PAIRSHA256Ristretto25519:
		pk.h = crypto.SHA256.New()
		if len(salt) != sha256SaltSize {
			return nil, ErrInvalidSaltSize
		}
		pk.salt = salt
	default:
		return nil, ErrInvalidPAIRMode
	}

	pk.scalar = ristretto255.NewScalar()
	if err := pk.scalar.UnmarshalText(scalar); err != nil {
		return nil, err
	}

	return pk, nil
}

// hash hashes the data using the private key's hash function with the salt.
func (pk *PrivateKey) hash(data []byte) []byte {
	// salt the hash function
	pk.h.Write(pk.salt)
	// hash the data
	pk.h.Write(data)
	return pk.h.Sum(nil)
}

// Encrypt first hashes the data with a salted hash function,
// it then derives the hashed data to an element of the group
// and encrypts it using the private key.
func (pk *PrivateKey) Encrypt(data []byte) ([]byte, error) {
	// hash the data
	data = pk.hash(data)

	// map hashed data to a point on the curve
	element := ristretto255.NewElement()
	uniformized := sha512.Sum512(data)
	element.FromUniformBytes(uniformized[:])

	// encrypt the data
	element.ScalarMult(pk.scalar, element)

	// return base64 encoded encrypted data
	return element.MarshalText()
}

// ReEncrypt re-encrypts the ciphertext using the same private key.
func (pk *PrivateKey) ReEncrypt(ciphertext []byte) ([]byte, error) {
	// unmarshal the ciphertext to an element of the group
	cipher := ristretto255.NewElement()
	if err := cipher.UnmarshalText(ciphertext); err != nil {
		return nil, err
	}

	// re-encrypt the group element by multiplying it with the private key
	cipher.ScalarMult(pk.scalar, cipher)

	return cipher.MarshalText()
}

// Decrypt undoes the encryption using the private key once, and returns the element of the group.
func (pk *PrivateKey) Decrypt(ciphertext []byte) ([]byte, error) {
	// unmarshal the ciphertext to an element of the group
	cipher := ristretto255.NewElement()
	if err := cipher.UnmarshalText(ciphertext); err != nil {
		return nil, err
	}

	// decrypt the group element by multiplying it with the inverse of the private key
	inverse := ristretto255.NewScalar()
	inverse.Invert(pk.scalar)

	cipher.ScalarMult(inverse, cipher)

	return cipher.MarshalText()
}
