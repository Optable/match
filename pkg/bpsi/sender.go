package bpsi

import (
	"context"
	"io"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/optable/match/internal/util"
)

// stage 1: load all local IDs into a bloom filter
// stage 2: serialize the bloomfilter out into rw

// Sender side of the BPSI protocol
type Sender struct {
	rw     io.ReadWriter
	logger logr.Logger
	bf     bloomfilter
}

// NewSender returns a bloomfilter sender initialized to
// use rw as the communication layer
func NewSender(ctx context.Context, rw io.ReadWriter) *Sender {
	// fetch and set up logger
	logger, err := logr.FromContext(ctx)
	if err != nil {
		logger = stdr.New(nil)
		// default logger with verbosity 0
		stdr.SetVerbosity(0)
	}
	logger = logger.WithValues("protocol", "bpsi")
	return &Sender{rw: rw, logger: logger}
}

// Send initiates a BPSI exchange
// that are read from identifiers, until identifiers closes.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) error {
	// pick a bloomfilter implementation
	s.bf, _ = NewBloomfilter(BloomfilterTypeBitsAndBloom, n)
	// stage 1: load all local IDs into a bloom filter
	stage1 := func() error {
		s.logger.V(1).Info("Starting stage 1")

		for id := range identifiers {
			s.bf.Add(id)
		}

		s.logger.V(1).Info("Finished stage 1")
		return nil
	}

	// stage 2: serialize the bloomfilter out into rw
	stage2 := func() error {
		s.logger.V(1).Info("Starting stage 2")
		_, err := s.bf.WriteTo(s.rw)

		s.logger.V(1).Info("Finished stage 2")
		return err
	}

	// run stage1
	if err := util.Sel(ctx, stage1); err != nil {
		return err
	}

	// run stage2
	if err := util.Sel(ctx, stage2); err != nil {
		return err
	}

	s.logger.V(1).Info("sender finished.")
	return nil
}
