package crypto

import (
	"github.com/zeebo/blake3"
)

// PseudorandomGenerate is a pseudorandom generator (PRG) using a
// deterministic random bit generator (DRBG) as specified by NIST
// Special Publication 800-90A Revision 1. Blake3 is used here.
func PseudorandomGenerate(dst []byte, seed []byte, h *blake3.Hasher) error {
	if len(dst) < len(seed) {
		copy(dst, seed)
		return nil
	}

	// reset internal state
	h.Reset()
	if _, err := h.Write(seed); err != nil {
		return err
	}

	drbg := h.Digest()

	_, err := drbg.Read(dst)

	return err
}
