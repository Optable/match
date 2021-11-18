package crypto

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/alecthomas/unsafeslice"
	"github.com/optable/match/internal/util"
	"github.com/twmb/murmur3"
	"github.com/zeebo/blake3"
)

// PseudorandomCode is implemented as follows:
// C(x) = AES(1||h(x)[:15]) ||
//        AES(2||h(x)[:15]) ||
//        AES(3||h(x)[:15]) ||
//        AES(4||h(x)[:15])
// where h() is the Murmur3 hashing function.
// PseudorandomCode is passed the src as well as the associated hash
// index. It also requires an AES block cipher.
// The full pseudorandom code consists of four 16 byte encrypted AES
// blocks that are encoded into a slice of 64 bytes. The hash function is
// constructed with the hash index as its two seeds. It is fed the full
// ID source. It returns two uint64s which are cast to a slice of bytes.
// The output is shifted right to allow prepending of the block index.
// For each block, the prepended value is changed to indicate the block
// index (1, 2, 3, 4) before being used as the source for the AES encode.
func PseudorandomCode(aesBlock cipher.Block, src []byte, hIdx byte) []byte {
	// prepare destination
	dst := make([]byte, aes.BlockSize*4)

	// hash id and the hash index
	lo, hi := murmur3.SeedSum128(uint64(hIdx), uint64(hIdx), src)

	// store in scratch slice
	s := unsafeslice.ByteSliceFromUint64Slice([]uint64{lo, hi})
	copy(s[1:], s) // shift for prepending

	// encrypt
	s[0] = 1
	aesBlock.Encrypt(dst[:aes.BlockSize], s)
	s[0] = 2
	aesBlock.Encrypt(dst[aes.BlockSize:aes.BlockSize*2], s)
	s[0] = 3
	aesBlock.Encrypt(dst[aes.BlockSize*2:aes.BlockSize*3], s)
	s[0] = 4
	aesBlock.Encrypt(dst[aes.BlockSize*3:], s)
	return dst
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

func Encrypt(key []byte, ind uint8, plaintext []byte) ([]byte, error) {
	return xorCipherWithBlake3(key, ind, plaintext)
}

func Decrypt(key []byte, ind uint8, ciphertext []byte) ([]byte, error) {
	return xorCipherWithBlake3(key, ind, ciphertext)
}
