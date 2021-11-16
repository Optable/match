package crypto

import (
	"bytes"
	"crypto/aes"
	"math/rand"
	"testing"
	"time"
)

var (
	p      = []byte("example testing plaintext that holds important secrets: %QWEQW$##%Y^&%^*(*)&, []m")
	aesKey = make([]byte, 16)
	xorKey = make([]byte, len(p))
	prng   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func init() {
	prng.Read(aesKey)
	prng.Read(xorKey)
}

func TestEncryptDecrypt(t *testing.T) {
	ciphertext, err := Encrypt(xorKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := Decrypt(xorKey, 1, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(p, plain) {
		t.Fatalf("decryption should not work!")
	}

	plain, err = Decrypt(xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(p, plain) {
		t.Fatalf("Decryption should have worked")
	}
}

func BenchmarkPseudorandomCode(b *testing.B) {
	// the normal input is a 64 byte digest with a byte indicating
	// which hash function is used to compute the cuckoo hash
	in := make([]byte, 64)
	prng.Read(in)
	var hIdx byte
	aesBlock, _ := aes.NewCipher(aesKey)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PseudorandomCode(aesBlock, in, hIdx)
	}
}

func BenchmarkEncrypt(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Encrypt(xorKey, 0, p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	c, err := Encrypt(xorKey, 0, p)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := Decrypt(xorKey, 0, c); err != nil {
			b.Fatal(err)
		}
	}
}
