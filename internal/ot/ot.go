package ot

import (
	"fmt"
	"io"
)

/*
OT interface
*/

var (
	ErrBaseCountMissMatch = fmt.Errorf("provided slices is not the same length as the number of base OT")
	ErrEmptyMessage       = fmt.Errorf("attempt to perform OT on empty messages")
)

// OT implements different BaseOT
type OT interface {
	Send(messages [][][]byte, rw io.ReadWriter) error
	Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error
}

// NewBaseOT returns an OT of type t
func NewBaseOT(baseCount int, msgLen []int) (OT, error) {
	return newNaorPinkas(baseCount, msgLen)
}
