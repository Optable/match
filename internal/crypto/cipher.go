package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

// CipherMode represents a particular cipher implementation chosen from
// the various cipher suite implementations in Go
type CipherMode int64

const (
	Unsupported CipherMode = iota
	GCM
	XORBlake3
)

const nonceSize = 12 //aesgcm NonceSize

// PseudorandomCode is implemented as follows:
// C(x) = AES(x[:16]) || AES(x[16:32]) || AES(x[32:48]) || AES(x[48:])
// src is padded to 64 bytes before being encrypted in blocks of 16 bytes.
// Blocks consisting only of padding are not encrypted. On success,
// PseudorandomCode returns an encrypted byte slice of 64 bytes.
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

// PseudorandomCodeWithHashIndex is implemented as follows:
// C(x) = AES(x[:16]) || AES(x[16:32]) || AES(x[32:48]) || AES(x[48:])
// PseudorandomCodeWithHashIndex is passed the src as well as the
// associated hash index. When padding the src to 64 bytes, if there
// is an empty byte, instead the hash index is placed there
// (effectively appending the hash index). Blocks of 16 bytes are then
// encrypted. Blocks consisting only of padding are not encrypted. On
// success, PseudorandomCodeWithHashIndex returns an encrypted byte
// slice of 64 bytes.
func PseudorandomCodeWithHashIndex(aesBlock cipher.Block, src []byte, hIdx byte) (dst []byte) {
	dst = make([]byte, aes.BlockSize*4)
	// effectively pad src
	copy(dst, src)
	if len(src) < aes.BlockSize*4 {
		dst[len(src)] = hIdx
	}

	// encrypt
	aesBlock.Encrypt(dst[:aes.BlockSize], dst[:aes.BlockSize])
	if len(src)+1 <= aes.BlockSize {
		return dst
	}

	aesBlock.Encrypt(dst[aes.BlockSize:aes.BlockSize*2], dst[aes.BlockSize:aes.BlockSize*2])
	if len(src)+1 <= aes.BlockSize*2 {
		return dst
	}

	aesBlock.Encrypt(dst[aes.BlockSize*2:aes.BlockSize*3], dst[aes.BlockSize*2:aes.BlockSize*3])
	if len(src)+1 <= aes.BlockSize*3 {
		return dst
	}

	aesBlock.Encrypt(dst[aes.BlockSize*3:], dst[aes.BlockSize*3:])
	return dst
}

// H(seed) xor src, where H is modeled as a pseudorandom generator.
func xorCipherWithPRG(s *blake3.Hasher, seed []byte, src []byte) (dst []byte, err error) {
	dst = make([]byte, len(src))
	s.Reset()
	if _, err := s.Write(seed); err != nil {
		return nil, err
	}
	d := s.Digest()
	if _, err := d.Read(dst); err != nil {
		return nil, err
	}
	err = util.ConcurrentBitOp(util.Xor, dst, src)
	return dst, err
}

// Blake3 has XOF which is perfect for doing xor cipher.
func xorCipherWithBlake3(key []byte, ind uint8, src []byte) ([]byte, error) {
	hash := make([]byte, len(src))
	err := getBlake3Hash(key, ind, hash)
	if err != nil {
		return nil, err
	}
	err = util.ConcurrentBitOp(util.Xor, hash, src)
	return hash, err
}

func getBlake3Hash(key []byte, ind uint8, dst []byte) error {
	h := blake3.New()
	if _, err := h.Write(key); err != nil {
		return err
	}
	if _, err := h.Write([]byte{ind}); err != nil {
		return err
	}

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

func Encrypt(mode CipherMode, key []byte, ind uint8, plaintext []byte) ([]byte, error) {
	switch mode {
	case Unsupported:
		return nil, fmt.Errorf("unsupported encrypt mode")
	case GCM:
		return gcmEncrypt(key, plaintext)
	case XORBlake3:
		return xorCipherWithBlake3(key, ind, plaintext)
	}

	return nil, fmt.Errorf("wrong encrypt mode")
}

func Decrypt(mode CipherMode, key []byte, ind uint8, ciphertext []byte) ([]byte, error) {
	switch mode {
	case Unsupported:
		return nil, fmt.Errorf("unsupported decrypt mode")
	case GCM:
		return gcmDecrypt(key, ciphertext)
	case XORBlake3:
		return xorCipherWithBlake3(key, ind, ciphertext)
	}

	return nil, fmt.Errorf("wrong decrypt mode")
}

// EncryptLen computes ciphertext length in bytes
func EncryptLen(mode CipherMode, msgLen int) int {
	switch mode {
	case Unsupported:
		return msgLen
	case GCM:
		return nonceSize + aes.BlockSize + msgLen
	case XORBlake3:
		fallthrough
	default:
		return msgLen
	}
}
