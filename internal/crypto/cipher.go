package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/alecthomas/unsafeslice"
	"github.com/optable/match/internal/util"
	"github.com/twmb/murmur3"
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
// C(x) = AES(1||h(x)[:15]) ||
//        AES(2||h(x)[:15]) ||
//        AES(3||h(x)[:15]) ||
//        AES(4||h(x)[:15])
// where h() is the Murmur3 hashing function.
// PseudorandomCode is passed the src as well as the associated hash
// index. It also requires an AES block cipher.
// The full pseudorandom code consists of four 16 byte encrypted AES
// blocks that are encoded into a slice of 64 bytes. During construction
// the last block (last 16 bytes) is used as a workspace.
// For each block, first the block index (1, 2, 3, 4) is placed at the
// 48th index (first element of the last block). The hash function is
// constructed with the hash index as its two seeds. It is fed the full
// ID source. It returns two uint64s which are cast to a slice of bytes
// of which the first 15 bytes are copied into the remainder of the last
// block (indices 49-64). Finally this block is used as the source for
// the AES encode and the destination is the actual proper block position.
func PseudorandomCode(aesBlock cipher.Block, src []byte, hIdx byte) (dst []byte, err error) {
	// prepare our destination
	dst = make([]byte, aes.BlockSize*4)
	dst[aes.BlockSize*3] = 1 // use last block as workspace to prepend block index

	// hash id and the hash index
	hi, lo := murmur3.SeedSum128(uint64(hIdx), uint64(hIdx), src)

	// copy into destination slice
	copy(dst[aes.BlockSize*3+1:], unsafeslice.ByteSliceFromUint64Slice([]uint64{hi, lo}))

	// encrypt
	aesBlock.Encrypt(dst[:aes.BlockSize], dst[aes.BlockSize*3:])
	dst[aes.BlockSize*3] = 2
	aesBlock.Encrypt(dst[aes.BlockSize:aes.BlockSize*2], dst[aes.BlockSize*3:])
	dst[aes.BlockSize*3] = 3
	aesBlock.Encrypt(dst[aes.BlockSize*2:aes.BlockSize*3], dst[aes.BlockSize*3:])
	dst[aes.BlockSize*3] = 4
	aesBlock.Encrypt(dst[aes.BlockSize*3:], dst[aes.BlockSize*3:])
	return dst, nil
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
