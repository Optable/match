package oprf

import "io"

/*
OPRF interface
*/

const (
	KKRT = iota
)

type OPRF interface {
	Send(rw io.ReadWriter) ([]key, error)
	Receive(choices [][]uint8, rw io.ReadWriter) ([][]byte, error)
	Encode(k key, in []byte) (out []byte, err error)
}
