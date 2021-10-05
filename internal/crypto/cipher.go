package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

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
// C(x) = AES(1||x) || AES(2||x) || AES(3||x) || AES(4||X)
// extracted in bits for the KKRT n choose 1 OPRF
// secretKey is a 16 byte slice for AES-128
// k is the desired number of bytes
// on success, pseudorandomCode returns a byte slice of length k.
func PseudorandomCodeDense(secretKey, src []byte) (dst []byte) {
	aesBlock, _ := aes.NewCipher(secretKey)
	dst = make([]byte, aes.BlockSize*4)

	// pad src
	input := pad(src)
	input[0] = 1

	// encrypt
	aesBlock.Encrypt(dst[:aes.BlockSize], input)
	input[0] = 2
	aesBlock.Encrypt(dst[aes.BlockSize:aes.BlockSize*2], input)
	input[0] = 3
	aesBlock.Encrypt(dst[aes.BlockSize*2:aes.BlockSize*3], input)
	input[0] = 4
	aesBlock.Encrypt(dst[aes.BlockSize*3:], input)

	// extract pseudorandom bytes to bits
	//util.ExtractBytesToBits(dst[aes.BlockSize*32:], dst[:aes.BlockSize*32])
	return dst
}

func PseudorandomCode(secretKey, src []byte) (dst []byte) {
	aesBlock, _ := aes.NewCipher(secretKey)
	dst = make([]byte, aes.BlockSize*4*9)

	// pad src
	input := pad(src)
	input[0] = 1

	// encrypt
	aesBlock.Encrypt(dst[aes.BlockSize*32:aes.BlockSize*33], input)
	input[0] = 2
	aesBlock.Encrypt(dst[aes.BlockSize*33:aes.BlockSize*34], input)
	input[0] = 3
	aesBlock.Encrypt(dst[aes.BlockSize*34:aes.BlockSize*35], input)
	input[0] = 4
	aesBlock.Encrypt(dst[aes.BlockSize*35:], input)

	// extract pseudorandom bytes to bits
	util.ExtractBytesToBits(dst[aes.BlockSize*32:], dst[:aes.BlockSize*32])
	return dst[:aes.BlockSize*32]
}

// pad aes block, with the first byte reserved for PseudorandomCode
func pad(src []byte) (tmp []byte) {
	tmp = make([]byte, len(src)+aes.BlockSize-len(src)%aes.BlockSize)
	copy(tmp[1:], src)
	return
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
