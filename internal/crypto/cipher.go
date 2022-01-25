package crypto

import (
	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

// XorCipherWithBlake3 uses the output of Blake3 XOF as pseudorandom
// bytes to perform a XOR cipher.
func XorCipherWithBlake3(key []byte, ind byte, src []byte) []byte {
	hash := make([]byte, len(src))
	getBlake3Hash(key, ind, hash)
	util.ConcurrentBitOp(util.Xor, hash, src)
	return hash
}

func getBlake3Hash(key []byte, ind byte, dst []byte) {
	h := blake3.New()
	h.Write(key)
	h.Write([]byte{ind})

	// convert to *digest to take a snapshot of the hashstate for XOF
	d := h.Digest()
	d.Read(dst)
}
