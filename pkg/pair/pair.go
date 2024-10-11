package pair

import (
	"crypto"
	"crypto/sha512"
	"errors"
	"hash"
	mrandv2 "math/rand/v2"

	"github.com/gtank/ristretto255"
)

type PAIRMode uint8

const (
	// PAIRSHA256Ristretto255 is PAIR with SHA256 as hash function and Ristretto255 as the group.
	PAIRSHA256Ristretto255 PAIRMode = 0x01
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
	mode PAIRMode

	// salt for h
	salt []byte

	// private key
	scalar *ristretto255.Scalar
}

// New instantiates a new private key with the given salt and scalar.
// It expects the scalar to be base64 encoded.
func (p PAIRMode) New(salt []byte, scalar []byte) (*PrivateKey, error) {
	pk := &PrivateKey{
		mode: p,
	}

	switch p {
	case PAIRSHA256Ristretto255:
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
	var h hash.Hash

	switch pk.mode {
	case PAIRSHA256Ristretto255:
		h = crypto.SHA256.New()
	default:
	}

	// salt the hash function
	h.Write(pk.salt)
	// hash the data
	h.Write(data)
	return h.Sum(nil)
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

// Shuffle shuffles the data in place by using the Fisher-Yates algorithm.
// Note that ideally, it should be called with less than 2^32-1 (4 billion) elements.
func Shuffle(data [][]byte) {
	// NOTE: since go 1.20, math.Rand seeds the global random number generator.
	// V2 uses ChaCha8 generator as the global one.
	mrandv2.Shuffle(len(data), func(i, j int) {
		data[i], data[j] = data[j], data[i]
	})
}
