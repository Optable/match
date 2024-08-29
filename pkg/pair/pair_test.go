package pair

import (
	"crypto/rand"
	"crypto/sha512"
	"strings"
	"testing"

	"github.com/gtank/ristretto255"
)

func TestPAIR(t *testing.T) {
	var (
		salt   = make([]byte, sha256SaltSize)
		scalar = ristretto255.NewScalar()
	)

	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}

	// sha512 produces a 64-byte psuedo-uniformized data
	src := sha512.Sum512(salt)
	scalar.FromUniformBytes(src[:])
	sk, err := scalar.MarshalText()
	if err != nil {
		t.Fatalf("failed to marshal the scalar: %s", err.Error())
	}

	// Create a new PAIR instance
	pairID := PAIRSHA256Ristretto25519

	pair, err := pairID.New(salt, sk)
	if err != nil {
		t.Fatalf("failed to instantiate a new PAIR instance: %s", err.Error())
	}

	var data = []byte("alice@hello.com")

	// Encrypt the data
	ciphertext, err := pair.Encrypt(data)
	if err != nil {
		t.Fatalf("failed to encrypt the data: %s", err.Error())
	}

	// Re-encrypt the data
	ciphertext2, err := pair.ReEncrypt(ciphertext)
	if err != nil {
		t.Fatalf("failed to re-encrypt the data: %s", err.Error())
	}

	// Decrypt the data
	decrypted, err := pair.Decrypt(ciphertext2)
	if err != nil {
		t.Fatalf("failed to decrypt the data: %s", err.Error())
	}

	if strings.Compare(string(ciphertext), string(decrypted)) != 0 {
		t.Fatalf("want: %s, got: %s", string(ciphertext), string(decrypted))
	}
}
