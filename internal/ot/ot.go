package ot

import (
	"crypto/rand"
	"errors"
	"io"

	"github.com/optable/match/internal/util"
)

/*
OT interface
*/

var (
	ErrBaseCountMissMatch = errors.New("provided slices is not the same length as the number of base OT")
	ErrEmptyMessage       = errors.New("attempt to perform OT on empty messages")
)

// OT implements different BaseOT
type OT interface {
	Send(messages []OTMessage, rw io.ReadWriter) error
	Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error
}

// OTMessage represent a pair of messages
// where an OT receiver with choice bit 0 will
// correctly decode the first message
// and an OT receiver with choice bit 1 will
// correclty decode the second message
type OTMessage [2][]byte

// SampleRandomOTMessage allocates a slice of OTMessage, each OTMessage contains a pair of messages.
// Extra elements are added to each column to be a multiple of 512. Every slice is filled with pseudorandom bytes
// values from a rand reader.
func SampleRandomOTMessages(rows, elems int) ([]OTMessage, error) {
	// instantiate matrix
	matrix := make([]OTMessage, rows)
	for row := range matrix {
		for col := range matrix[row] {
			matrix[row][col] = make([]byte, (elems+util.PadTill512(elems))/8)
			// fill
			if _, err := rand.Read(matrix[row][col]); err != nil {
				return nil, err
			}
		}
	}

	return matrix, nil

}
