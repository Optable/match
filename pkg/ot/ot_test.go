package ot

import (
	"crypto/aes"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestInitCurve(t *testing.T) {
	curveTests := []struct {
		name string
		want string
	}{
		{"P224", "P-224"},
		{"P256", "P-256"},
		{"P384", "P-384"},
		{"P521", "P-521"},
	}

	for _, tt := range curveTests {
		c, _ := initCurve(tt.name)
		got := c.Params().Name
		if got != tt.want {
			t.Errorf("InitCurve(%s): want curve %s, got curve %s", tt.name, tt.name, got)
		}
	}
}

func TestDeriveKey(t *testing.T) {
	c := elliptic.P256()
	_, px, py, err := elliptic.GenerateKey(c, rand.Reader)
	if err != nil {
		t.Errorf("elliptic curve GenerateKey failed: %s", err)
	}

	p := elliptic.Marshal(c, px, py)
	key := deriveKey(p)
	if len(key) != 32 {
		t.Errorf("Derived key length is not 32, got: %d", len(key))
	}
}

func TestEncrypDecrypt(t *testing.T) {
	c := elliptic.P256()
	_, px, py, err := elliptic.GenerateKey(c, rand.Reader)
	if err != nil {
		t.Errorf("elliptic curve GenerateKey failed: %s", err)
	}

	p := elliptic.Marshal(c, px, py)
	key := deriveKey(p)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Errorf("Error instantiating aes block cipher: %s\n", err)
	}

	plaintext := []byte("example testing plaintext with special chars: %QWEQW$##%Y^&%^*(*)&, []")
	ciphertext, err := encrypt(block, plaintext)
	if err != nil {
		t.Errorf("Error when encrypt: %s\n", err)
	}

	plain, err := decrypt(block, ciphertext)
	if err != nil {
		t.Errorf("Error when decrypt: %s\n", err)
	}

	if len(plaintext) != len(plain) {
		t.Errorf("Decrypted plaintext is not the same lenghth as plaintext used in encryption: %d, %d", len(plaintext), len(plain))

	}

	equal := true
	for i, b := range plaintext {
		if b != plain[i] {
			equal = false
		}
	}

	if !equal {
		t.Errorf("plaintext used in encrption is not the same as the decrypted plaintext.")
	}
}
