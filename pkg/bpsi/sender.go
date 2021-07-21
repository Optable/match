package bpsi

import (
	"context"
	"io"

	"github.com/devopsfaith/bloomfilter"
	baseBloomfilter "github.com/devopsfaith/bloomfilter/bloomfilter"
	"github.com/optable/match/internal/util"
)

// stage 1: load all local IDs into a bloom filter
// stage 2: serialize the bloomfilter out into rw

// Sender side of the NPSI protocol
type Sender struct {
	rw io.ReadWriter
	bf *baseBloomfilter.Bloomfilter
}

// NewSender returns a bloomfilter sender initialized to
// use rw as the communication layer
func NewSender(rw io.ReadWriter) *Sender {
	return &Sender{rw: rw}
}

// Send initiates a BPSI exchange
// that are read from identifiers, until identifiers closes.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) error {
	// create the bloom filter
	s.bf = baseBloomfilter.New(bloomfilter.Config{N: (uint)(n), P: 0.5, HashName: bloomfilter.HASHER_OPTIMAL})
	// stage 1: load all local IDs into a bloom filter
	stage1 := func() error {
		for id := range identifiers {
			s.bf.Add(id)
		}
		return nil
	}

	// stage 2: serialize the bloomfilter out into rw
	stage2 := func() error {
		if b, err := s.bf.MarshalBinary(); err == nil {
			if _, err := s.rw.Write(b); err != nil {
				return err
			}
			return nil
		} else {
			return err
		}
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

func (s *Sender) sendAll(identifiers <-chan []byte) error {
	return nil
}
