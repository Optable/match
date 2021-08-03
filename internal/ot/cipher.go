package ot

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

/*
Various cipher suite implementation in golang
*/

const (
	CTR = iota
	GCM
	XORBlake2
	XORBlake3
	XORShake
)

// pad aes block, no need for unpad since we only need to encrypt
// and not decrypt the aes blocks.
func pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

// pseudorandomCode is implemented as follows:
// C(x) = AES(1||x) || AES(2||x) || AES(3||x) || AES(4||X)
// extracted in bits for the KKRT n choose 1 OPRF
// secretKey is a 16 byte slice for AES-128
// k is the desired number of bytes
// on success, pseudorandomCode returns a byte slice of length k.
func pseudorandomCode(secretKey []byte, k int, src []byte) []byte {
	block, _ := aes.NewCipher(secretKey)
	tmp := make([]byte, aes.BlockSize*4)
	dst := make([]byte, aes.BlockSize*4*8)

	// pad src
	src = pad(src)

	// encrypt
	block.Encrypt(tmp[:aes.BlockSize], append([]byte{1}, src...))
	block.Encrypt(tmp[aes.BlockSize:aes.BlockSize*2], append([]byte{2}, src...))
	block.Encrypt(tmp[aes.BlockSize*2:aes.BlockSize*3], append([]byte{3}, src...))
	block.Encrypt(tmp[aes.BlockSize*3:], append([]byte{4}, src...))

	// extract pseudorandom bytes to bits
	util.ExtractBytesToBits(tmp, dst)
	// return desired number of bytes
	return dst[:k]
}

// H(seed) xor src, where H is modeled as a pseudorandom generator.
func xorCipherWithPRG(s *blake3.Hasher, seed []byte, src []byte) (dst []byte, err error) {
	dst = make([]byte, len(src))
	s.Reset()
	s.Write(seed)
	d := s.Digest()
	d.Read(dst)
	return util.XorBytes(src, dst)
}

// Blake3 has XOF which is perfect for doing xor cipher.
func xorCipherWithBlake3(key []byte, ind uint8, src []byte) (dst []byte, err error) {
	hash := make([]byte, len(src))
	err = getBlake3Hash(key, ind, hash)
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
	if err != nil {
		return err
	}

	return nil
}

// Shake from the Sha3 family has XOF which is perfect for doing xor cipher.
func xorCipherWithShake(key []byte, ind uint8, src []byte) (dst []byte, err error) {
	hash := make([]byte, len(src))
	getShakeHash(key, ind, hash)
	return util.XorBytes(hash, src)
}

func getShakeHash(key []byte, ind uint8, dst []byte) {
	h := sha3.NewShake256()
	h.Write(key)
	h.Write([]byte{ind})
	h.Read(dst)
}

// xorCipher returns the result of H(ind, key) XOR src
// note that encrypt and decrypt in XOR cipher are the same.
func xorCipherWithBlake2(key []byte, ind uint8, src []byte) (dst []byte, err error) {
	hash := make([]byte, len(src))
	err = getBlake2Hash(key, ind, hash)
	if err != nil {
		return nil, err
	}

	return util.XorBytes(hash, src)
}

// getHash produce hash digest of the key and index
func getBlake2Hash(key []byte, ind uint8, dst []byte) (err error) {
	d, err := blake2b.NewXOF(uint32(len(dst)), nil)
	if err != nil {
		return err
	}

	d.Write(key)
	d.Write([]byte{ind})
	d.Read(dst)

	return
}

// aes CTR + HMAC encrypt decrypt
func ctrEncrypt(key []byte, plaintext []byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	l := aes.BlockSize + len(plaintext)
	ciphertext = make([]byte, l+32)
	if _, err := rand.Read(ciphertext[:aes.BlockSize]); err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, ciphertext[:aes.BlockSize])
	stream.XORKeyStream(ciphertext[aes.BlockSize:l], plaintext)

	h := sha3.NewShake256()
	// reuse IV as key for mac
	h.Write(ciphertext[:aes.BlockSize])
	h.Write(ciphertext[aes.BlockSize:l])
	h.Read(ciphertext[l:])
	return
}

func ctrDecrypt(key []byte, ciphertext []byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv, c, mac := ciphertext[:aes.BlockSize], ciphertext[aes.BlockSize:len(ciphertext)-32], ciphertext[len(ciphertext)-32:]
	plaintext = make([]byte, len(c))
	stream := cipher.NewCTR(block, iv)

	// verify mac
	mac2 := make([]byte, 32)
	h := sha3.NewShake256()
	h.Write(iv)
	h.Write(c)
	h.Read(mac2)
	if !bytes.Equal(mac, mac2) {
		return nil, fmt.Errorf("cipher text is not authenticated")
	}
	stream.XORKeyStream(plaintext, c)
	return
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

func encrypt(mode int, key []byte, ind uint8, plaintext []byte) ([]byte, error) {
	switch mode {
	case CTR:
		return ctrEncrypt(key, plaintext)
	case GCM:
		return gcmEncrypt(key, plaintext)
	case XORBlake2:
		return xorCipherWithBlake2(key, ind, plaintext)
	case XORBlake3:
		return xorCipherWithBlake3(key, ind, plaintext)
	case XORShake:
		return xorCipherWithShake(key, ind, plaintext)
	}

	return nil, fmt.Errorf("wrong encrypt mode")
}

func decrypt(mode int, key []byte, ind uint8, ciphertext []byte) ([]byte, error) {
	switch mode {
	case CTR:
		return ctrDecrypt(key, ciphertext)
	case GCM:
		return gcmDecrypt(key, ciphertext)
	case XORBlake2:
		return xorCipherWithBlake2(key, ind, ciphertext)
	case XORBlake3:
		return xorCipherWithBlake3(key, ind, ciphertext)
	case XORShake:
		return xorCipherWithShake(key, ind, ciphertext)
	}

	return nil, fmt.Errorf("wrong decrypt mode")
}

// compute ciphertext length in bytes
func encryptLen(mode int, msgLen int) int {
	switch mode {
	case CTR:
		return aes.BlockSize + msgLen
	case GCM:
		return nonceSize + aes.BlockSize + msgLen
	case XORBlake2, XORBlake3, XORShake:
		fallthrough
	default:
		return msgLen
	}
}
