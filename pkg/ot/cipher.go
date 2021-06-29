package ot

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"

	"golang.org/x/crypto/blake2b"
)

const (
	AES = iota
	XOR
)

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
	buf := make([]byte, binary.MaxVarintLen16)
	binary.PutUvarint(buf, uint64(ind))
	h.Write(buf)
	hash = h.Sum(nil)

	return
}

// aes GCM block cipher encryption
func blockCipherEncrypt(key []byte, plaintext []byte) (ciphertext []byte, err error) {
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

func blockCipherDecrypt(key []byte, ciphertext []byte) (plaintext []byte, err error) {
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
	case AES:
		return blockCipherEncrypt(key, plaintext)
	case XOR:
		fallthrough
	default:
		return xorCipher(key, ind, plaintext)
	}
}

func decrypt(mode int, key []byte, ind uint8, ciphertext []byte) ([]byte, error) {
	switch mode {
	case AES:
		return blockCipherDecrypt(key, ciphertext)
	case XOR:
		fallthrough
	default:
		return xorCipher(key, ind, ciphertext)
	}
}

// compute ciphertext length in bytes
func encryptLen(mode int, msgLen int) int {
	switch mode {
	case AES:
		return nonceSize + aes.BlockSize + msgLen
	case XOR:
		fallthrough
	default:
		return msgLen
	}
}
