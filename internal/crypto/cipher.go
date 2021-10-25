package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"hash"

	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

/*
Various cipher suite implementation in golang
*/

const (
	GCM = iota
	XORBlake3

	nonceSize = 12 //aesgcm NonceSize
)

// PseudorandomCode is implemented as follows:
// C(x) = AES(x[:16]) || AES(x[16:32]) || AES(x[32:48]) || AES(x[48:])
// secretKey is a 16 byte slice for AES-128
// on success, pseudorandomCode returns a byte slice of 64 bytes.
func PseudorandomCode(aesBlock cipher.Block, src []byte) (dst []byte) {
	dst = make([]byte, aes.BlockSize*4)
	// effectively pad src
	copy(dst, src)

	// encrypt
	aesBlock.Encrypt(dst[:aes.BlockSize], dst[:aes.BlockSize])
	if len(src) <= aes.BlockSize {
		return dst
	}

	aesBlock.Encrypt(dst[aes.BlockSize:aes.BlockSize*2], dst[aes.BlockSize:aes.BlockSize*2])
	if len(src) <= aes.BlockSize*2 {
		return dst
	}

	aesBlock.Encrypt(dst[aes.BlockSize*2:aes.BlockSize*3], dst[aes.BlockSize*2:aes.BlockSize*3])
	if len(src) <= aes.BlockSize*3 {
		return dst
	}

	aesBlock.Encrypt(dst[aes.BlockSize*3:], dst[aes.BlockSize*3:])
	return dst
}

// PseudorandomCodeHmacSHA256 is implemented as follows:
// C(x) = HmacSHA256_key(x[:32]) || HmacSHA256_key(x[32:]
// secretKey is a 32 byte slice for Hmac-SHA256
// on success, pseudorandomCode returns a byte slice of 64 bytes.
func PseudorandomCodeHmacSHA256(prf hash.Hash, src []byte) (dst []byte) {
	dst = make([]byte, 64)

	// PRF
	prf.Write(src[:32])
	copy(dst[:32], prf.Sum(nil))

	prf.Write(src[32:])
	copy(dst[32:], prf.Sum(nil))

	return dst
}

// H(seed) xor src, where H is modeled as a pseudorandom generator.
func XorCipherWithPRG(s *blake3.Hasher, seed []byte, src []byte) (dst []byte, err error) {
	dst = make([]byte, len(src))
	s.Reset()
	s.Write(seed)
	d := s.Digest()
	d.Read(dst)
	return util.XorBytes(src, dst)
}

// Blake3 has XOF which is perfect for doing xor cipher.
func xorCipherWithBlake3(key []byte, ind uint8, src []byte) ([]byte, error) {
	hash := make([]byte, len(src))
	err := getBlake3Hash(key, ind, hash)
	if err != nil {
		return nil, err
	}
	return util.XorBytes(hash, src)
}

func getBlake3Hash(key []byte, ind uint8, dst []byte) error {
	h := blake3.New()
	h.Write(key)
	h.Write([]byte{ind})

	// convert to *digest to take a snapshot of the hashstate for XOF
	d := h.Digest()
	_, err := d.Read(dst)
	return err
}

// aes GCM block encryption decryption
func gcmEncrypt(key []byte, plaintext []byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// encrypted cipher text is appended after nonce
	ciphertext = aesgcm.Seal(nonce, nonce, plaintext, nil)
	return
}

func gcmDecrypt(key []byte, ciphertext []byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	nonce, enc := ciphertext[:nonceSize], ciphertext[nonceSize:]

	if plaintext, err = aesgcm.Open(nil, nonce, enc, nil); err != nil {
		return nil, err
	}
	return
}

func Encrypt(mode int, key []byte, ind uint8, plaintext []byte) ([]byte, error) {
	switch mode {
	case GCM:
		return gcmEncrypt(key, plaintext)
	case XORBlake3:
		return xorCipherWithBlake3(key, ind, plaintext)
	}

	return nil, fmt.Errorf("wrong encrypt mode")
}

func Decrypt(mode int, key []byte, ind uint8, ciphertext []byte) ([]byte, error) {
	switch mode {
	case GCM:
		return gcmDecrypt(key, ciphertext)
	case XORBlake3:
		return xorCipherWithBlake3(key, ind, ciphertext)
	}

	return nil, fmt.Errorf("wrong decrypt mode")
}

// compute ciphertext length in bytes
func EncryptLen(mode int, msgLen int) int {
	switch mode {
	case GCM:
		return nonceSize + aes.BlockSize + msgLen
	case XORBlake3:
		fallthrough
	default:
		return msgLen
	}
}
