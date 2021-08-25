package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"math/rand"
	"testing"
	"time"

	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

var (
	p      = []byte("example testing plaintext that holds important secrets: %QWEQW$##%Y^&%^*(*)&, []m")
	aesKey = make([]byte, 16)
	xorKey = make([]byte, len(p))
	r      = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func init() {
	r.Read(aesKey)
	r.Read(xorKey)
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

func TestCTREncrypDecrypt(t *testing.T) {
	ciphertext, err := Encrypt(CTR, aesKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := Decrypt(CTR, aesKey, 0, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(p, plain) {
		t.Errorf("error in decrypt, want: %s, got: %s", string(p), string(plain))
	}
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

func TestXORShakeEncryptDecrypt(t *testing.T) {
	ciphertext, err := Encrypt(XORShake, xorKey, 0, p)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := Decrypt(XORShake, xorKey, 1, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(p, plain) {
		t.Fatalf("decryption should not work!")
	}

	plain, err = Decrypt(XORShake, xorKey, 0, ciphertext)
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

	plain, err := Decrypt(XORBlake2, xorKey, 1, ciphertext)
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

func TestXORBytes(t *testing.T) {
	a := make([]byte, 32)
	r.Read(a)

	b := make([]byte, 32)
	r.Read(b)
	c, err := util.XorBytes(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(c, a) || bytes.Equal(c, b) {
		t.Fatalf("c should not be equal to a nor b")
	}

	c, err = util.XorBytes(a, c)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(c, b) {
		t.Fatalf("c should be equal to b")
	}
}

func TestPseudorandomGeneratorWithBlake3(t *testing.T) {
	seed := make([]byte, 424)
	r.Read(seed)
	n := 212
	p, _ := PseudorandomGeneratorWithBlake3(blake3.New(), seed, n)
	if bytes.Equal(make([]byte, n), p) {
		t.Fatalf("pseudorandom should not be 0")
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

func BenchmarkXORCipherWithPRG(b *testing.B) {
	s := blake3.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		XorCipherWithPRG(s, xorKey, p)
	}
}

func BenchmarkXORCipherWithAESCTR(b *testing.B) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xorCipherWithAESCTR(block, xorKey, p)
	}
}

func BenchmarkXORCipherWithAESCTR2(b *testing.B) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xorCipherWithAESCTR2(block, xorKey, p)
	}
}

func BenchmarkPRGWithBlake3(b *testing.B) {
	s := blake3.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PseudorandomGeneratorWithBlake3(s, xorKey, len(p))
	}
}

func BenchmarkPRGWithAESGCM(b *testing.B) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		b.Log(err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		b.Log(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pseudorandomGeneratorWithAESGCM(aesgcm, xorKey, len(p))
	}
}
