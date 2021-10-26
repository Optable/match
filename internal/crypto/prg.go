package crypto

import (
	"fmt"
	"math/rand"

	"github.com/zeebo/blake3"
)

// pseudorandom generator (PRG) using deterministic random bit generator (DRBG)
// AES ctr_drbg, and hmac_drbg
// as specified by NIST Special Publication 800-90A Revision 1.

const (
	MrandDrbg = iota
	AESCtrDrbg
	HashDrbg
	HmacDrbg
	HashXOF
)

var (
	ErrUnknownPRG     = fmt.Errorf("unknown pseudorandom generators")
	ErrNotImplemented = fmt.Errorf("drbg not implemented")
	ErrDstTooShort    = fmt.Errorf("destination is shorter than seed")
)

func PseudorandomGenerate(drbg int, seed []byte, length int) ([]byte, error) {
	// need expand?
	if length < len(seed) {
		return seed[:length], nil
	}

	switch drbg {
	case MrandDrbg:
		return prgWithSeed(seed, length), nil
	case AESCtrDrbg:
		return aesCTRDrbg(seed, length), nil
	case HashDrbg:
		return nil, ErrNotImplemented
	case HmacDrbg:
		// tried oasis Hmac drpg, which is 6 times slower than that of AES and math rand prg
		// so we are dropping this
		return nil, ErrNotImplemented
	case HashXOF:
		return blake3XOF(seed, length)
	}

	return nil, ErrUnknownPRG
}

// prgWithSeed returns a pseudorandom byte slice read from
// math rand seeded with seed.
func prgWithSeed(seed []byte, length int) (dst []byte) {
	dst = make([]byte, length)
	var source int64
	for i := 0; i < len(seed)/64; i++ {
		var s int64
		for j, b := range seed[i*64 : (i+1)*64] {
			s += (int64(b) << j)
		}
		source ^= s
	}

	r := rand.New(rand.NewSource(source))
	r.Read(dst)

	return dst
}

// aesCTRDrbg returns a pseudorandom byte slice generated using block cipher DRBG
// with AES in CTR mode.
func aesCTRDrbg(seed []byte, length int) (dst []byte) {
	var newSeed [48]byte
	copy(newSeed[:], seed)
	drbg := NewDRBG(&newSeed)

	dst = make([]byte, length)
	drbg.Fill(dst)

	return dst
}

// blake3XOF returns a byte slice generated with a seeded blake3 digest object.
func blake3XOF(seed []byte, length int) (dst []byte, err error) {
	h := blake3.New()
	h.Write(seed)
	drbg := h.Digest()

	dst = make([]byte, length)
	_, err = drbg.Read(dst)

	return dst, err
}

func PseudorandomGenerateWithBlake3XOF(dst []byte, seed []byte, h *blake3.Hasher) error {
	// need expand?
	if len(dst) < len(seed) {
		return ErrDstTooShort
	}

	h.Write(seed)
	drbg := h.Digest()

	_, err := drbg.Read(dst)

	return err
}
