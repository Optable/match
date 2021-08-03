package oprf

import "io"

/*
OPRF interface
*/

const (
	KKRT = iota
)

// OT implements different BaseOT
type OPRF interface {
	Send(rw io.ReadWriter) ([]key, error)
	Receive(choices [][]uint8, rw io.ReadWriter) ([][]byte, error)
}
