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
