package dhpsi

import (
	"context"
	"fmt"
	"io"

	"github.com/optable/match/internal/permutations"
	"github.com/optable/match/internal/util"
)

// (receiver, publisher: high cardinality) stage1: reads the identifiers from the sender, encrypt them and index them in a map
// (receiver, publisher: high cardinality) stage2.1: permute and write the local identifiers to the sender
// (receiver, publisher: high cardinality) stage2.2: reads back the identifiers from the sender and learns the intersection

// Receiver represents the receiver in a DHPSI operation, often the publisher.
// The receiver learns the intersection of matchable between its set and the set
// of the sender
type Receiver struct {
	rw io.ReadWriter
}

// NewReceiver returns a receiver initialized to
// use rw as the communication layer
func NewReceiver(rw io.ReadWriter) *Receiver {
	return &Receiver{rw: rw}
}

type permuted struct {
	position   int64
	identifier []byte
}

// IntersectFromReader on n matchables,
// sourced from r, returning the matching intersection.
// The format of an indentifier is
//  string\r\n
func (s *Receiver) IntersectFromReader(ctx context.Context, n int64, r io.Reader) ([][]byte, error) {
	var identifiers = exhaust(n, r)
	return s.Intersect(ctx, n, identifiers)
}

// Intersect on n matchables,
// sourced from identifiers, returning the matching intersection.
// The format of an indentifier is
//  string
func (s *Receiver) Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error) {
	// state
	var remoteIDs = make(map[[EncodedLen]byte]bool) // single write goroutine access from stage1
	var localIDs = make([][]byte, n)
	var receiverIDs = make(chan permuted)
	var matchedIDs = make(chan int64)
	// the final intersection
	var intersection [][]byte
	// the permutations algo used
	// it might contains a seed
	// or a pre-computed order so extract it for use
	// in the intersection part
	var permutations permutations.Permutations

	// pick a ristretto implementation
	gr, _ := NewRistretto(RistrettoTypeR255)
	// step1 : reads the identifiers from the sender, encrypt them and index the encoded ristretto point in a map
	stage1 := func() error {
		if reader, err := NewMultiplyParallelReader(s.rw, gr); err != nil {
			return err
		} else {
			for {
				// read
				var p [EncodedLen]byte
				if err := reader.Read(&p); err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
				// index
				remoteIDs[p] = true
			}
		}
	}
	// stage2.1 : permute and write the local identifiers to the sender
	stage21 := func() error {
		if writer, err := NewDeriveMultiplyParallelShuffler(s.rw, n, gr); err != nil {
			return err
		} else {
			// take a snapshot of the reverse of the permutations
			permutations = writer.Permutations()
			// read n identifiers from src
			// and
			//  1. index them locally
			//  2. write them to the sender
			var i int64
			for identifier := range identifiers {
				// save this input
				receiverIDs <- permuted{i, identifier} // {0, "0"}
				if err := writer.Shuffle(identifier); err != nil {
					return err
				}
				i++
			}

			return nil
		}
	}
	// step3: reads back the identifiers from the sender and learns the intersection
	stage22 := func() error {
		if reader, err := NewReader(s.rw); err != nil {
			return err
		} else {
			for i := int64(0); i < reader.Max(); i++ {
				// read
				var p [EncodedLen]byte
				if err := reader.Read(&p); err != nil {
					return fmt.Errorf("stage2.2: %v", err)
				}
				if remoteIDs[p] {
					// we can match this local identifier with one received
					// from the sender
					matchedIDs <- i
				}
			}
		}
		return nil
	}

	// run stage1
	if err := util.Sel(ctx, stage1); err != nil {
		return nil, err
	}
	// run stage2.1/2.2
	var done = 2
	var errs = util.Sels(stage21, stage22)
	for done != 0 {
		select {
		case err := <-errs:
			if err == nil {
				done--
			} else {
				return intersection, err
			}

		case <-ctx.Done():
			return intersection, ctx.Err()

		case pos := <-matchedIDs:
			intersection = append(intersection, localIDs[permutations.Shuffle(pos)])

		case p := <-receiverIDs:
			localIDs[p.position] = p.identifier
		}
	}

	return intersection, nil
}
