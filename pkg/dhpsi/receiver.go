package dhpsi

import (
	"bufio"
	"context"
	"io"
)

// (receiver, publisher: high cardinality) step1: reads the matchables from the receiver, encrypt them and index them in a map
// (receiver, publisher: high cardinality) step2: permute and write the local matchables to the sender
// (receiver, publisher: high cardinality) step3: reads back the matchables from the sender and learns the intersection

// Receiver represents the receiver in a DHPSI operation, often the publisher.
// The receiver learns the intersection of matchable between its set and the set
// of the sender
type Receiver struct {
	rw       io.ReadWriter
	receiver map[[EncodedLen]byte]bool
}

// NewReceiver returns a receiver initialized to
// use rw as the communication layer
func NewReceiver(rw io.ReadWriter) *Receiver {
	return &Receiver{rw: rw, receiver: make(map[[EncodedLen]byte]bool)}
}

// Intersect on n matchables,
// sourced from r, returning the matching intersection.
func (s *Receiver) Intersect(ctx context.Context, n int64, r io.Reader) ([][]byte, error) {
	// state
	var matchables [][]byte
	var matched [][]byte
	var permutations []int64
	// pick a ristretto implementation
	gr := NewRistretto(RistrettoTypeGR)
	// wrap src in a bufio reader
	src := bufio.NewReader(r)
	// step1 : reads the matchables from the receiver, encrypt them and index them in a map
	s1 := func() error {
		if r, err := NewReader(s.rw); err != nil {
			return err
		} else {
			for {
				// read, encrypt & index
				var p [EncodedLen]byte
				if err := r.Read(&p); err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
				p = gr.Multiply(p)
				s.receiver[p] = true
			}
		}
	}
	// step2 : permute and write the local matchables to the sender
	s2 := func() error {
		if s2encoder, err := NewShufflerEncoder(s.rw, n, gr); err != nil {
			return err
		} else {
			// take a snapshot of the permutations
			permutations = s2encoder.Permutations()
			// read N matchables from r
			// and write them to stage1
			for i := int64(0); i < n; i++ {
				line, err := SafeReadLine(src)
				if len(line) != 0 {
					// save this input
					matchables = append(matchables, line)
					if err := s2encoder.Encode(line); err != nil {
						return err
					}
				}
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
	// step3: reads back the matchables from the sender and learns the intersection
	s3 := func() error {
		if r, err := NewReader(s.rw); err != nil {
			return err
		} else {
			for i := int64(0); i < r.Max(); i++ {
				// read, encrypt & index
				var p [EncodedLen]byte
				if err := r.Read(&p); err != nil {
					return err
				}
				if s.receiver[p] {
					matched = append(matched, matchables[permutations[i]])
				}
			}
		}
		return nil
	}

	// run step1
	if err := sel(ctx, s1); err != nil {
		return matched, err
	}
	// run step2
	if err := sel(ctx, s2); err != nil {
		return matched, err
	}
	// run step3
	if err := sel(ctx, s3); err != nil {
		return matched, err
	}

	return matched, nil
}
