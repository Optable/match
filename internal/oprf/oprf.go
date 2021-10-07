package oprf

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

/*
OPRF interface
*/

const (
	k = 512 // width of base OT binary matrix as well as the pseudorandom code output length

	KKRT = iota
	ImprvKKRT
)

var ErrUnknownOPRF = fmt.Errorf("cannot create an OPRF that follows an unknown protocol")

type OPRF interface {
	Send(rw io.ReadWriter) ([]Key, error)
	Receive(choices [][]uint8, rw io.ReadWriter) ([][]byte, error)
}

// NewOPRF returns an OPRF of type t
func NewOPRF(t, m, baseOT int) (OPRF, error) {
	switch t {
	case KKRT:
		return newKKRT(m, baseOT, false)
	case ImprvKKRT:
		return newImprovedKKRT(m, baseOT, crypto.HashXOF, false)
	default:
		return nil, ErrUnknownOPRF
	}
}

// Key contains the relaxed OPRF key: (C, s), (j, q_j)
// the index j is implicit when key is stored into a key slice.
// Pseudorandom code C is represented by sk
type Key struct {
	sk []byte // secret key for pseudorandom code
	s  []byte // secret choice bits
	q  []byte // m x k bit matrice
}

// Encode computes and returns OPRF(k, in)
func (k Key) Encode(aesBlock cipher.Block, in []byte) (out []byte, err error) {
	// compute q_i ^ (C(r) & s)
	out = crypto.PseudorandomCode(aesBlock, in)

	if err = util.ConcurrentInPlaceAndBytes(out, k.s); err != nil {
		return nil, err
	}

	err = util.ConcurrentInPlaceXorBytes(out, k.q)
	return
}

func GetAesBlock(k Key) (cipher.Block, error) {
	return aes.NewCipher(k.sk)
}
