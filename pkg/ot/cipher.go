package ot

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

const (
	CTR = iota
	GCM
	XOR
)

// Since shake can hash t
//variable length hash digest, let's use it as a PRG oracle.

// xorBytes xors each byte from a with b and returns dst
// if a and b are the same length
func xorBytes(a, b []byte) (dst []byte, err error) {
	n := len(b)
	if n != len(a) {
		return nil, ErrByteLengthMissMatch
	}

	dst = make([]byte, n)

	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}

	return
}

// Shake from the Sha3 family produce variable length hash digest
// perfect for doing xor cipher.
func xorCipherWithShake(key []byte, ind uint8, src []byte) (dst []byte, err error) {
	hash := make([]byte, len(src))
	shakeHash(key, ind, hash)
	return xorBytes(hash, src)
}

func shakeHash(key []byte, ind uint8, dst []byte) {
	h := sha3.NewShake256()
	h.Write(key)
	h.Write([]byte{ind})
	h.Read(dst)
}

// xorCipher returns the result of H(ind, key) XOR src
// note that encrypt and decrypt in XOR cipher are the same.
func xorCipher(key []byte, ind uint8, src []byte) (dst []byte, err error) {
	// make sure we deal with plaintext less than hashDigest size
	n := len(src)

	hash, err := getHash(key, ind)
	if err != nil {
		return nil, err
	}

	if n > blake2b.Size {
		for len(hash) < n {
			hash = append(hash, hash[:]...)
		}
	}

	return xorBytes(hash[:n], src)
}

// getHash produce hash digest of the key and index
func getHash(key []byte, ind uint8) (hash []byte, err error) {
	h, err := blake2b.New512(nil)
	if err != nil {
		return nil, err
	}

	h.Write(key)
	h.Write([]byte{ind})
	hash = h.Sum(nil)

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
	if bytes.Compare(mac, mac2) != 0 {
		return nil, fmt.Errorf("Cipher text is not authenticated.")
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
	case XOR:
		fallthrough
	default:
		return xorCipher(key, ind, plaintext)
	}
}

func decrypt(mode int, key []byte, ind uint8, ciphertext []byte) ([]byte, error) {
	switch mode {
	case CTR:
		return ctrDecrypt(key, ciphertext)
	case GCM:
		return gcmDecrypt(key, ciphertext)
	case XOR:
		fallthrough
	default:
		return xorCipher(key, ind, ciphertext)
	}
}

// compute ciphertext length in bytes
func encryptLen(mode int, msgLen int) int {
	switch mode {
	case CTR:
		return aes.BlockSize + msgLen
	case GCM:
		return nonceSize + aes.BlockSize + msgLen
	case XOR:
		fallthrough
	default:
		return msgLen
	}
}
