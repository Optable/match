package ot

import (
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

var plaintext = []byte("example testing plaintext that holds important secrets: %QWEQW$##%Y^&%^*(*)&, []")

func TestBlockCipherEncrypDecrypt(t *testing.T) {
	c := elliptic.P256()
	_, px, py, err := elliptic.GenerateKey(c, rand.Reader)
	if err != nil {
		t.Fatalf("elliptic curve GenerateKey failed: %s", err)
	}

	p := elliptic.Marshal(c, px, py)
	key := deriveKey(p)

	ciphertext, err := encrypt(AES, key, 0, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(AES, key, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if len(plaintext) != len(plain) {
		t.Fatalf("error in decrypt, want %d len bytes, got %d len bytes", len(plaintext), len(plain))

	}

	equal := true
	for i, b := range plaintext {
		if b != plain[i] {
			equal = false
		}
	}

	if !equal {
		t.Errorf("error in decrypt, want: %s, got: %s", string(plaintext), string(plain))
	}
}

func TestXORCipherEncryptDecrypt(t *testing.T) {
	n := len(plaintext)
	key := make([]byte, n)
	rand.Read(key)

	ciphertext, err := encrypt(XOR, key, 0, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(XOR, key, 1, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) == string(plaintext) {
		t.Fatalf("decryption should not work!")
	}

	plain, err = decrypt(XOR, key, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if string(plain) != string(plaintext) {
		t.Fatalf("Decryption should have worked")
	}
}

func TestXORBytes(t *testing.T) {
	a := make([]byte, 32)
	rand.Read(a)

	b := make([]byte, 32)
	rand.Read(b)
	c, err := xorBytes(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if string(c) == string(a) || string(c) == string(b) {
		t.Fatalf("c should not be equal to a nor b")
	}

	c, err = xorBytes(a, c)
	if err != nil {
		t.Fatal(err)
	}
	if string(c) != string(b) {
		t.Fatalf("c should be equal to b")
	}
}

func BenchmarkXORCipherEncrypt(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		encrypt(XOR, key, 0, plaintext)
	}
}

func BenchmarkXORCipherDecrypt(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	c, _ := encrypt(XOR, key, 0, plaintext)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		decrypt(XOR, key, 0, c)
	}
}

func BenchmarkAESCipherEncrypt(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		encrypt(AES, key, 0, plaintext)
	}
}

func BenchmarkAESCipherDecrypt(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	c, _ := encrypt(AES, key, 0, plaintext)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		decrypt(AES, key, 0, c)
	}
}
