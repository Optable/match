package oprf

import "io"

/*
OPRF interface
*/

const (
	KKRT = iota
)

type OPRF interface {
	Send(rw io.ReadWriter) ([]Key, error)
	Receive(choices [][]uint8, rw io.ReadWriter) ([][]byte, error)
	Encode(k Key, in []byte) (out []byte, err error)
}

// Key contains the relaxed OPRF key: (C, s), (j, q_j)
// the index j is implicit when key is stored into a key slice.
// Pseudorandom code C is represented by sk
type Key struct {
	sk []byte // secret key for pseudorandom code
	s  []byte // secret choice bits
	q  []byte // m x k bit matrice
}
