package oprf

import (
	"crypto/aes"
	"errors"
	"io"

	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/util"
)

// width of base OT binary matrix  as well as the ouput
// length of PseudorandomCode (in bits)
const k = aes.BlockSize * 4 * 8

var ErrUnknownOPRF = errors.New("cannot create an OPRF that follows an unknown protocol")

type OPRF interface {
	Send(rw io.ReadWriter) (Key, error)
	Receive(choices *cuckoo.Cuckoo, sk []byte, rw io.ReadWriter) ([cuckoo.Nhash]map[uint64]uint64, error)
}

// NewOPRF returns an OPRF of type t
func NewOPRF(m int) (OPRF, error) {
	return newImprovedKKRT(m)
}

// Key contains the relaxed OPRF key: (C, s), (j, q_j)
// Pseudorandom code C is represented by aes block seeded with s
// q stores the received OT extension matrix chosen with secret
// seed s.
type Key struct {
	s []byte   // secret choice bits
	q [][]byte // m x k bit matrice
}

// Encode computes and returns OPRF(k, in)
func (k Key) Encode(j uint64, pseudorandomBytes []byte) error {
	return util.ConcurrentDoubleBitOp(util.AndXor, pseudorandomBytes, k.s, k.q[j])
}
