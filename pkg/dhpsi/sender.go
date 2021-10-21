package dhpsi

import (
	"context"
	"fmt"
	"io"

	"github.com/optable/match/internal/util"
	"github.com/optable/match/pkg/log"
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

// SendFromReader initiates a DHPSI exchange with n identifiers
// that are read from r. The format of an indentifier is
//  string\n
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e\r\n
func (s *Sender) SendFromReader(ctx context.Context, n int64, r io.Reader) error {
	// extract r into a channel via SafeRead
	var identifiers = util.Exhaust(n, r)
	return s.Send(ctx, n, identifiers)
}

// Send initiates a DHPSI exchange with n identifiers
// that are read from the identifiers channel, until identifiers closes or n is reached.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) error {
	// fetch logger
	var logger = log.GetLoggerFromContextWithName(ctx, "dhpsi")

	// pick a ristretto implementation
	gr, _ := NewRistretto(RistrettoTypeR255)
	// stage1 : writes the permutated identifiers to the receiver
	stage1 := func() error {
		logger.Info("Starting stage 1")

		writer, err := NewDeriveMultiplyParallelShuffler(s.rw, n, gr)
		if err != nil {
			return err
		}
		// read N matchables from r
		// and write them to stage1
		// shuffle will error out if more than N
		// are read from identifiers
		for identifier := range identifiers {
			if err := writer.Shuffle(identifier); err != nil {
				return err
			}
		}

		logger.Info("Finished stage 1")
		return nil
	}

	// stage2 : reads the identifiers from the receiver, encrypt them and send them back
	stage2 := func() error {
		logger.Info("Starting stage 2")

		reader, err := NewMultiplyParallelReader(s.rw, gr)
		if err != nil {
			return err
		}
		writer, err := NewWriter(s.rw, reader.Max())
		if err != nil {
			return err
		}
		for i := int64(0); i < reader.Max(); i++ {
			var p [EncodedLen]byte
			if err := reader.Read(&p); err != nil {
				if err != io.EOF {
					return err
				}
			}
			if err := writer.Write(p); err != nil {
				return fmt.Errorf("stage2: %v", err)
			}
		}

		logger.Info("Finished stage 2")
		return nil
	}

	// run stage1
	if err := util.Sel(ctx, stage1); err != nil {
		return err
	}
	// run stage2
	if err := util.Sel(ctx, stage2); err != nil {
		return err
	}

	return nil
}
