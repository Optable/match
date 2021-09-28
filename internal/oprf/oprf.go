package oprf

import (
	"io"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

/*
OPRF interface
*/

const (
	KKRT = iota
)

type OPRF interface {
	Send(rw io.ReadWriter) ([]Key, error)
	Receive(choices [][]uint8, rw io.ReadWriter) ([][]byte, error)
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
func (k Key) Encode(in []byte) (out []byte, err error) {
	// compute q_i ^ (C(r) & s)
	out = crypto.PseudorandomCode(k.sk, in)
	err = util.InPlaceAndBytesBits(k.s, out)
	if err != nil {
		return nil, err
	}
	err = util.InPlaceXorBytes(k.q, out)
	return
}
