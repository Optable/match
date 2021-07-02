package ot

import (
	"bytes"
	"testing"

	//"golang.org/x/crypto/sha3"
	"github.com/zeebo/blake3"
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

func TestXORwithPRG(t *testing.T) {
	h := blake3.New()
	seed := []byte{1, 1, 0, 1, 0, 1, 1, 0}
	src := []byte{0, 0, 1, 0, 1, 0, 0, 1}
	d, _ := xorCipherWithPRG(h, seed, src)
	e, _ := xorCipherWithPRG(h, seed, d)
	if bytes.Compare(src, e) != 0 {
		t.Logf("xor is not equal!")
	}
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

	if bytes.Compare(p, plain) != 0 {
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

	if bytes.Compare(p, plain) != 0 {
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

	if bytes.Compare(p, plain) == 0 {
		t.Fatalf("decryption should not work!")
	}

	plain, err = decrypt(XORShake, xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(p, plain) != 0 {
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

	if bytes.Compare(p, plain) == 0 {
		t.Fatalf("decryption should not work!")
	}

	plain, err = xorCipherWithBlake2(xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(p, plain) != 0 {
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

	if bytes.Compare(p, plain) == 0 {
		t.Fatalf("decryption should not work!")
	}

	plain, err = decrypt(XORBlake3, xorKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(p, plain) != 0 {
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
