package dhpsi

import (
	"bufio"
	"context"
	"fmt"
	"io"
)

// operations
// (sender, often advertiser: low cardinality) stage1: writes the permutated identifiers to the receiver
// (sender, oftem advertiser: low cardinality) stage2: reads the identifiers from the receiver, encrypt them and send them back

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

// Send initiates a DHPSI exchange with n identifiers
// that are read from r. The format of an indentifier is
//  PREFIX:MATCHABLE\r\n
// example:
//  e:0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e\r\n
func (s *Sender) Send(ctx context.Context, n int64, r io.Reader) error {
	// pick a ristretto implementation
	gr, _ := NewRistretto(RistrettoTypeR255)
	// wrap src in a bufio reader
	src := bufio.NewReader(r)
	// stage1 : writes the permutated identifiers to the receiver
	stage1 := func() error {
		if writer, err := NewDeriveMultiplyParallelShuffler(s.rw, n, gr); err != nil {
			return err
		} else {
			// read N matchables from r
			// and write them to stage1
			for i := int64(0); i < n; i++ {
				line, err := SafeReadLine(src)
				// some data was returned
				if len(line) != 0 {
					if err := writer.Shuffle(line); err != nil {
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
	// stage2 : reads the identifiers from the receiver, encrypt them and send them back
	stage2 := func() error {
		reader, err := NewMultiplyParallelReader(s.rw, gr)
		if err != nil {
			return err
		}
		if writer, err := NewWriter(s.rw, reader.Max()); err != nil {
			return err
		} else {
			for i := int64(0); i < reader.Max(); i++ {
				var p [EncodedLen]byte
				if err := reader.Multiply(&p); err != nil {
					if err != io.EOF {
						return err
					}
				}
				if err := writer.Write(p); err != nil {
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

/*
	for {
		// read one batch
		var points = make([][EncodedLen]byte, batchSize)
		n, err := reader.Multiplies(points)
		// process data
		if n != 0 {
			for i := 0; i < n; i++ {
				if err := writer.Write(points[i]); err != nil {
					return fmt.Errorf("stage2: %v", err)
				}
			}
		}
		// process errors
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("stage2: %v", err)
			} else if n == 0 {
				break
			}
		}
	}
	return nil
*/
