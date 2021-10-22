package oprf

import (
	"crypto/cipher"
	"fmt"
	"io"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/cuckoo"
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
	Send(rw io.ReadWriter) (Key, error)
	//Receive(choices [][]uint8, rw io.ReadWriter) ([][]byte, error)
	Receive(choices *cuckoo.Cuckoo, rw io.ReadWriter) ([][]byte, error)
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
// Pseudorandom code C is represented by aes block seeded with s
// q stores the received OT extension matrix chosen with secret
// seed s.
type Key struct {
	block cipher.Block
	s     []byte   // secret choice bits
	q     [][]byte // m x k bit matrice
}

// Encode computes and returns OPRF(k, in)
func (k Key) Encode(j uint64, in []byte) (out []byte, err error) {
	// compute q_i ^ (C(r) & s)
	out = crypto.PseudorandomCode(k.block, in)

	if err = util.ConcurrentInPlaceAndBytes(out, k.s); err != nil {
		return nil, err
	}

	err = util.ConcurrentInPlaceXorBytes(out, k.q[j])
	return
}
