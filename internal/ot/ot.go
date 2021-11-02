package ot

import (
	"fmt"
	"io"

	"github.com/optable/match/internal/crypto"
)

/*
OT interface
*/

const (
	// enumerated base OTs
	NaorPinkas = iota
	Simplest
)

var (
	ErrUnknownOT          = fmt.Errorf("cannot create an Ot that follows an unknown protocol")
	ErrBaseCountMissMatch = fmt.Errorf("provided slices is not the same length as the number of base OT")
	ErrEmptyMessage       = fmt.Errorf("attempt to perform OT on empty messages")
)

// OT implements different BaseOT
type OT interface {
	Send(messages [][][]byte, rw io.ReadWriter) error
	Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error
}

// NewBaseOT returns an OT of type t
func NewBaseOT(t int, ristretto bool, baseCount int, curveName string, msgLen []int, cipherMode crypto.CipherMode) (OT, error) {
	switch t {
	case NaorPinkas:
		if ristretto {
			return newNaorPinkasRistretto(baseCount, msgLen, cipherMode)
		}
		return newNaorPinkas(baseCount, curveName, msgLen, cipherMode)
	case Simplest:
		if ristretto {
			return newSimplestRistretto(baseCount, msgLen, cipherMode)
		}
		return newSimplest(baseCount, curveName, msgLen, cipherMode)
	default:
		return nil, ErrUnknownOT
	}
}
