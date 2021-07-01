package ot

import (
	"bytes"
	"testing"
)

var (
	plaintext = []byte("example testing plaintext that holds important secrets: %QWEQW$##%Y^&%^*(*)&, []m")
	aesKey    = make([]byte, 16)
	xorKey    = make([]byte, len(plaintext))
)

func init() {
	r.Read(aesKey)
	r.Read(xorKey)
}

func TestCTREncrypDecrypt(t *testing.T) {
	ciphertext, err := encrypt(CTR, aesKey, 0, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(CTR, aesKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(plaintext, plain) != 0 {
		t.Errorf("error in decrypt, want: %s, got: %s", string(plaintext), string(plain))
	}
}

func TestGCMEncrypDecrypt(t *testing.T) {
	ciphertext, err := encrypt(GCM, aesKey, 0, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(GCM, aesKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(plaintext, plain) != 0 {
		t.Errorf("error in decrypt, want: %s, got: %s", string(plaintext), string(plain))
	}
}

func TestXORCipherWithShakeEncryptDecrypt(t *testing.T) {
	ciphertext, err := xorCipherWithShake(xorKey, 0, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := xorCipherWithShake(xorKey, 1, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(plaintext, plain) == 0 {
		t.Fatalf("decryption should not work!")
	}

	plain, err = xorCipherWithShake(xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(plaintext, plain) != 0 {
		t.Fatalf("Decryption should have worked")
	}
}

func TestXORCipherEncryptDecrypt(t *testing.T) {
	ciphertext, err := encrypt(XOR, xorKey, 0, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(XOR, xorKey, 1, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(plaintext, plain) == 0 {
		t.Fatalf("decryption should not work!")
	}

	plain, err = decrypt(XOR, xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(plaintext, plain) != 0 {
		t.Fatalf("Decryption should have worked")
	}
}

func TestXORBytes(t *testing.T) {
	a := make([]byte, 32)
	r.Read(a)

	b := make([]byte, 32)
	r.Read(b)
	c, err := xorBytes(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(c, a) == 0 || bytes.Compare(c, b) == 0 {
		t.Fatalf("c should not be equal to a nor b")
	}

	c, err = xorBytes(a, c)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(c, b) != 0 {
		t.Fatalf("c should be equal to b")
	}
}

func BenchmarkXORCipherWithShakeEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xorCipherWithShake(xorKey, 0, plaintext)
	}
}

func BenchmarkXORCipherWithShakeDecrypt(b *testing.B) {
	c, err := xorCipherWithShake(xorKey, 0, plaintext)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		xorCipherWithShake(xorKey, 0, c)
	}
}

func BenchmarkXORCipherEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xorCipher(xorKey, 0, plaintext)
	}
}

func BenchmarkXORCipherDecrypt(b *testing.B) {
	c, err := xorCipher(xorKey, 0, plaintext)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		xorCipher(xorKey, 0, c)
	}
}

func BenchmarkAesGcmEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		gcmEncrypt(aesKey, plaintext)
	}
}

func BenchmarkAesGcmDecrypt(b *testing.B) {
	c, _ := gcmEncrypt(aesKey, plaintext)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gcmDecrypt(aesKey, c)
	}
}

func BenchmarkAesCtrEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctrEncrypt(aesKey, plaintext)
	}
}

func BenchmarkAesCtrDecrypt(b *testing.B) {
	c, err := ctrEncrypt(aesKey, plaintext)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctrDecrypt(aesKey, c)
	}
}
