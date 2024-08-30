package pair

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"slices"
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
	pairID := PAIRSHA256Ristretto255

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

func genData(n int) [][]byte {
	data := make([][]byte, n)
	for i := 0; i < n; i++ {
		// marshaled ristretto255.Scalar is 44 bytes
		data[i] = make([]byte, 44)
		rand.Read(data[i])
	}
	return data
}

func TestShuffle(t *testing.T) {
	data := genData(1 << 10) // 1k
	orig := make([][]byte, len(data))
	copy(orig, data)

	// shuffle the data in place
	Shuffle(data)

	once := make([][]byte, len(data))
	copy(once, data)

	if slices.EqualFunc(data, orig, bytes.Equal) {
		t.Fatalf("data not shuffled")
	}

	// shuffle again
	Shuffle(data)

	if slices.EqualFunc(data, once, bytes.Equal) {
		t.Fatalf("data not shuffled")
	}
}

func BenchmarkShuffleOneMillionIDs(b *testing.B) {
	data := genData(1 << 20) // 1m
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Shuffle(data)
	}
}
