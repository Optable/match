package kkrtpsi

import (
	"context"
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/util"
)

// stage 1: samples 3 hash seeds and sends them to receiver for cuckoo hash
// stage 2: OPRF Send
// stage 3: read local IDs and compute OPRF(k, id) and send them to receiver.

// Sender side of the KKRTPSI protocol
type Sender struct {
	rw io.ReadWriter
}

// NewSender returns a KKRTPSI sender initialized to
// use rw as the communication layer
func NewSender(rw io.ReadWriter) *Sender {
	return &Sender{rw: rw}
}

// Send initiates a KKRTPSI exchange
// that are read from identifiers, until identifiers closes.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) error {
	// stage 1: sample 3 hash seeds and write them to receiver
	var seeds [cuckoo.Nhash][]byte
	stage1 := func() error {
		// init randomness source
		rand.Seed(time.Now().UnixNano())
		// sample Nhash hash seeds
		for i := range seeds {
			seeds[i] = make([]byte, hash.SaltLength)
			rand.Read(seeds[i])
			// write it into rw
			if _, err := s.rw.Write(seeds[i]); err != nil {
				return err
			}
		}
		return nil
	}

	// stage 2:
	stage2 := func() error {
		return nil
	}

	// stage 3:
	stage3 := func() error {
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

	// run stage3
	if err := util.Sel(ctx, stage3); err != nil {
		return err
	}

	return nil
}
