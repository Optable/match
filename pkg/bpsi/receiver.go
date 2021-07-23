package bpsi

import (
	"context"
	"fmt"
	"io"

	"github.com/optable/match/internal/util"
)

// ErrReadingBloomfilter is triggered if there's an IO problem reading the remote side bloomfilter structure
var ErrReadingBloomfilter = fmt.Errorf("could not read a bloomfilter structure from the remote end")

// stage 1: read the bloomfilter from the remote side
// stage 2: read local IDs and compare with the remote bloomfilter
//          to produce intersections

// Receiver side of the BPSI protocol
type Receiver struct {
	rw io.ReadWriter
}

// NewReceiver returns a bloomfilter receiver initialized to
// use rw as the communication layer
func NewReceiver(rw io.ReadWriter) *Receiver {
	return &Receiver{rw: rw}
}

// Intersect on matchables read from the identifiers channel,
// returning the matching intersection, using the NPSI protocol.
// The format of an indentifier is
//  string
func (r *Receiver) Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error) {
	var bf bloomfilter
	var intersected [][]byte

	// stage 1: read the bloomfilter from the remote side
	stage1 := func() error {
		if _bf, _, err := ReadFrom(r.rw); err == nil {
			bf = _bf
			return nil
		} else {
			return err
		}
	}

	// stage 2: read local IDs and compare with the remote bloomfilter
	//          to produce intersections
	stage2 := func() error {
		for identifier := range identifiers {
			if bf.Check(identifier) {
				intersected = append(intersected, identifier)
			}
		}
		return nil
	}

	// run stage1
	if err := util.Sel(ctx, stage1); err != nil {
		return intersected, err
	}

	// run stage2
	if err := util.Sel(ctx, stage2); err != nil {
		return intersected, err
	}

	return intersected, nil
}
