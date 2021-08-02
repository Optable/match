package ot

import (
	"bytes"
	"testing"
)

var (
	p      = []byte("example testing plaintext that holds important secrets: %QWEQW$##%Y^&%^*(*)&, []m")
	aesKey = make([]byte, 16)
	xorKey = make([]byte, len(p))
)

func init() {
	r.Read(aesKey)
	r.Read(xorKey)
}

func TestCTREncrypDecrypt(t *testing.T) {
	ciphertext, err := encrypt(CTR, aesKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(CTR, aesKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(p, plain) {
		t.Errorf("error in decrypt, want: %s, got: %s", string(p), string(plain))
	}
}

func TestGCMEncrypDecrypt(t *testing.T) {
	ciphertext, err := encrypt(GCM, aesKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(GCM, aesKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(p, plain) {
		t.Errorf("error in decrypt, want: %s, got: %s", string(p), string(plain))
	}
}

func TestXORShakeEncryptDecrypt(t *testing.T) {
	ciphertext, err := encrypt(XORShake, xorKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(XORShake, xorKey, 1, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(p, plain) {
		t.Fatalf("decryption should not work!")
	}

	plain, err = decrypt(XORShake, xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(p, plain) {
		t.Fatalf("Decryption should have worked")
	}
}

func TestXORBlake2EncryptDecrypt(t *testing.T) {
	ciphertext, err := xorCipherWithBlake2(xorKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(XORBlake2, xorKey, 1, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(p, plain) {
		t.Fatalf("decryption should not work!")
	}

	plain, err = xorCipherWithBlake2(xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(p, plain) {
		t.Fatalf("Decryption should have worked")
	}
}

func TestXORBlake3EncryptDecrypt(t *testing.T) {
	ciphertext, err := encrypt(XORBlake3, xorKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := decrypt(XORBlake3, xorKey, 1, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(p, plain) {
		t.Fatalf("decryption should not work!")
	}

	plain, err = decrypt(XORBlake3, xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(p, plain) {
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
	if bytes.Equal(c, a) || bytes.Equal(c, b) {
		t.Fatalf("c should not be equal to a nor b")
	}

	c, err = xorBytes(a, c)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(c, b) {
		t.Fatalf("c should be equal to b")
	}
}

func BenchmarkXORCipherWithShakeEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xorCipherWithShake(xorKey, 0, p)
	}
}

func BenchmarkXORCipherWithShakeDecrypt(b *testing.B) {
	c, err := xorCipherWithShake(xorKey, 0, p)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		xorCipherWithShake(xorKey, 0, c)
	}
}

func BenchmarkXORCipherWithBlake2Encrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xorCipherWithBlake2(xorKey, 0, p)
	}
}

func BenchmarkXORCipherWithBlake2Decrypt(b *testing.B) {
	c, err := xorCipherWithBlake2(xorKey, 0, p)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		xorCipherWithBlake2(xorKey, 0, c)
	}
}

func BenchmarkXORCipherWithBlake3Encrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xorCipherWithBlake3(xorKey, 0, p)
	}
}

func BenchmarkXORCipherWithBlake3Decrypt(b *testing.B) {
	c, err := xorCipherWithBlake3(xorKey, 0, p)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		xorCipherWithBlake3(xorKey, 0, c)
	}
}

func BenchmarkAesGcmEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		gcmEncrypt(aesKey, p)
	}
}

func BenchmarkAesGcmDecrypt(b *testing.B) {
	c, _ := gcmEncrypt(aesKey, p)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gcmDecrypt(aesKey, c)
	}
}

func BenchmarkAesCtrEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctrEncrypt(aesKey, p)
	}
}

func BenchmarkAesCtrDecrypt(b *testing.B) {
	c, err := ctrEncrypt(aesKey, p)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctrDecrypt(aesKey, c)
	}
}
