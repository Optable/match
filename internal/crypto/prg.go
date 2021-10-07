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
	ErrUnknownPRG     = fmt.Errorf("cannot create unknown pseudorandom generators")
	ErrNotImplemented = fmt.Errorf("drbg not implemented")
)

func PseudorandomGenerate(drbg int, seed []byte, length int) ([]byte, error) {
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

func prgWithSeed(seed []byte, length int) (dst []byte) {
	// need expand?
	if length < len(seed) {
		return seed[:length]
	}
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

func aesCTRDrbg(seed []byte, length int) (dst []byte) {
	// need expand?
	if length < len(seed) {
		return seed[:length]
	}

	var newSeed [48]byte
	copy(newSeed[:], seed)
	drbg := NewDRBG(&newSeed)

	dst = make([]byte, length)
	drbg.Fill(dst)

	return dst
}

func blake3XOF(seed []byte, length int) (dst []byte, err error) {
	// need expand?
	if length < len(seed) {
		return seed[:length], nil
	}

	h := blake3.New()
	h.Write(seed)
	drbg := h.Digest()

	dst = make([]byte, length)
	_, err = drbg.Read(dst)

	return dst, err
}
