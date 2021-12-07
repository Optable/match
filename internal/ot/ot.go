package ot

import (
	"errors"
	"io"
)

/*
OT interface
*/

var (
	ErrBaseCountMissMatch = errors.New("provided slices is not the same length as the number of base OT")
	ErrEmptyMessage       = errors.New("attempt to perform OT on empty messages")
)

// OT implements a BaseOT
type OT interface {
	Send(messages []OTMessage, rw io.ReadWriter) error
	Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error
}

// OTMessage represent a pair of messages
// where an OT receiver with choice bit 0 will
// correctly decode the first message
// and an OT receiver with choice bit 1 will
// correctly decode the second message
type OTMessage [2][]byte
