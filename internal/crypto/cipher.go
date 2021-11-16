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
// blocks that are encoded into a slice of 64 bytes. During construction
// the last block (last 16 bytes) is used as a workspace.
// For each block, first the block index (1, 2, 3, 4) is placed at the
// 48th index (first element of the last block). The hash function is
// constructed with the hash index as its two seeds. It is fed the full
// ID source. It returns two uint64s which are cast to a slice of bytes
// of which the first 15 bytes are copied into the remainder of the last
// block (indices 49-64). Finally this block is used as the source for
// the AES encode and the destination is the actual proper block position.
func PseudorandomCode(aesBlock cipher.Block, src []byte, hIdx byte) []byte {
	// prepare our destination
	dst := make([]byte, aes.BlockSize*4)
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
