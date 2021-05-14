package dhpsi

import (
	"bufio"
	"context"
	"fmt"
	"io"
)

// operations
// (sender, often advertiser: low cardinality) stage1: writes the permutated matchables to the receiver
// (sender, oftem advertiser: low cardinality) stage2: reads the matchables from the receiver, encrypt them and send them back

// Sender represents the sender in a DHPSI operation, often the advertiser.
// The sender initiates the transfer and in the case of DHPSI, it learns nothing.
type Sender struct {
	rw io.ReadWriter
}

// NewSender returns a sender initialized to
// use rw as the communication layer
func NewSender(rw io.ReadWriter) *Sender {
	return &Sender{rw: rw}
}

// Send initiates a DHPSI exchange with n matchables
// that are read from r. The format of a matchable is
//  PREFIX:MATCHABLE\r\n
// example:
//  e:0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e\r\n
func (s *Sender) Send(ctx context.Context, n int64, r io.Reader) error {
	// pick a ristretto implementation
	gr := NewRistretto(RistrettoTypeGR)
	// wrap src in a bufio reader
	src := bufio.NewReader(r)
	// stage1 : writes the permutated matchables to the receiver
	stage1 := func() error {
		if s1encoder, err := NewDeriveMultiplyDirectEncoder(s.rw, n, gr); err != nil {
			return err
		} else {
			// read N matchables from r
			// and write them to stage1
			for i := int64(0); i < n; i++ {
				line, err := SafeReadLine(src)
				// some data was returned
				if len(line) != 0 {
					if err := s1encoder.Encode(line); err != nil {
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
	// stage2 : reads the matchables from the receiver, encrypt them and send them back
	stage2 := func() error {
		step2reader, err := NewReader(s.rw)
		if err != nil {
			return err
		}
		if step2encoder, err := NewMultiplyEncoder(s.rw, step2reader.Max(), gr); err != nil {
			return err
		} else {
			for i := int64(0); i < step2reader.Max(); i++ {
				var p [EncodedLen]byte
				if err := step2reader.Read(&p); err != nil {
					if err != io.EOF {
						return err
					}
				}
				if err := step2encoder.Encode(p); err != nil {
					return fmt.Errorf("stage2: %v", err)
				}
			}
			return nil
		}
	}

	// run stage1
	if err := sel(ctx, stage1); err != nil {
		return err
	}
	// run stage2
	if err := sel(ctx, stage2); err != nil {
		return err
	}

	return nil
}
