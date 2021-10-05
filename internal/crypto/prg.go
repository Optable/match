package crypto

import (
	"fmt"
	"math/rand"

	"github.com/optable/match/internal/util"
)

// pseudorandom generator (PRG) using deterministic random bit generator (DRBG)
// AES ctr_drbg, and hmac_drbg
// as specified by NIST Special Publication 800-90A Revision 1.

const (
	MrandDrbg = iota
	AESCtrDrbg
	AESCtrDrbgDense
	HashDrbg
	HmacDrbg
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
	case AESCtrDrbgDense:
		return aesCTRDrbgDense(seed, length), nil
	case HashDrbg:
		return nil, ErrNotImplemented
	case HmacDrbg:
		// tried oasis Hmac drpg, which is 6 times slower than that of AES and math rand prg
		// so we are dropping this
		return nil, ErrNotImplemented
	}

	return nil, ErrUnknownPRG
}

func prgWithSeed(seed []byte, length int) (dst []byte) {
	// need expand?
	if length < len(seed) {
		return seed[:length]
	}

	tmp := make([]byte, (length+7)/8)
	dst = make([]byte, len(tmp)*8)
	var source int64
	for i := 0; i < len(seed)/64; i++ {
		var s int64
		for j, b := range seed[i*64 : (i+1)*64] {
			s += (int64(b) << j)
		}
		source ^= s
	}

	r := rand.New(rand.NewSource(source))
	r.Read(tmp)

	// extract pseudorandom bytes to bits
	util.ExtractBytesToBits(tmp, dst)
	return dst[:length]
}

func aesCTRDrbg(seed []byte, length int) (dst []byte) {
	// need expand?
	if length < len(seed) {
		return seed[:length]
	}

	var newSeed [48]byte
	copy(newSeed[:], seed)
	drbg := NewDRBG(&newSeed)

	tmp := make([]byte, (length+7)/8)
	dst = make([]byte, len(tmp)*8)
	drbg.Fill(tmp)

	// extract pseudorandom bytes to bits
	util.ExtractBytesToBits(tmp, dst)
	return dst[:length]
}

func aesCTRDrbgDense(seed []byte, length int) (dst []byte) {
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
