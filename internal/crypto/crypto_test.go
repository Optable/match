package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/sha256"
	"math/rand"
	"testing"
	"time"

	"github.com/zeebo/blake3"
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

func TestGCMEncrypDecrypt(t *testing.T) {
	ciphertext, err := Encrypt(GCM, aesKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := Decrypt(GCM, aesKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(p, plain) {
		t.Errorf("error in decrypt, want: %s, got: %s", string(p), string(plain))
	}
}

func TestXORBlake3EncryptDecrypt(t *testing.T) {
	ciphertext, err := Encrypt(XORBlake3, xorKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := Decrypt(XORBlake3, xorKey, 1, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(p, plain) {
		t.Fatalf("decryption should not work!")
	}

	plain, err = Decrypt(XORBlake3, xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(p, plain) {
		t.Fatalf("Decryption should have worked")
	}
}

func BenchmarkSha(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sha256.Sum256(p)
	}
}

func BenchmarkBlake3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		blake3.Sum256(p)
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
		_, err := PseudorandomCode(aesBlock, in, hIdx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXORCipherWithBlake3Encrypt(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := xorCipherWithBlake3(xorKey, 0, p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXORCipherWithBlake3Decrypt(b *testing.B) {
	c, err := xorCipherWithBlake3(xorKey, 0, p)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := xorCipherWithBlake3(xorKey, 0, c); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAesGcmEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := gcmEncrypt(aesKey, p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAesGcmDecrypt(b *testing.B) {
	c, _ := gcmEncrypt(aesKey, p)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := gcmDecrypt(aesKey, c); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXORCipherWithPRG(b *testing.B) {
	s := blake3.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := xorCipherWithPRG(s, xorKey, p); err != nil {
			b.Fatal(err)
		}
	}
}
