package npsi

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/optable/match/pkg/log"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/util"
)

// stage 1: receive a random salt K from P1
// stage 2: send hashes salted with K to P1

// Sender represents sender side of the NPSI protocol
type Sender struct {
	rw io.ReadWriter
}

// NewSender returns a sender initialized to
// use rw as the communication layer
func NewSender(rw io.ReadWriter) *Sender {
	return &Sender{rw: rw}
}

// Send initiates a NPSI exchange
// that are read from identifiers, until identifiers closes.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) error {
	// fetch logger
	var logger = log.GetLoggerFromContextWithName(ctx, "npsi")

	// hold k
	var k = make([]byte, hash.SaltLength)
	// stage 1: receive a random salt K from P1
	stage1 := func() error {
		logger.Info("Starting stage 1")
		logger.V(1).Info("Testing this logger")
		if n, err := s.rw.Read(k); err != nil {
			return fmt.Errorf("stage1: %v", err)
		} else if n != hash.SaltLength {
			return hash.ErrSaltLengthMismatch
		}

		logger.Info("Finished stage 1")
		return nil
	}

	// stage 2: send hashes salted with K to P1
	stage2 := func() error {
		logger.Info("Starting stage 2")
		logger.V(2).Info("Testing this logger with verbosity = 2")
		// get a hasher
		h, err := hash.New(hash.Highway, k)
		if err != nil {
			return err
		}
		// inform the receiver of the size
		// its about to receive
		if err := binary.Write(s.rw, binary.BigEndian, &n); err != nil {
			return err
		}
		// make a channel to receive local x,h pairs
		sender := HashAllParallel(h, identifiers)
		// exhaust the hashes into the receiver
		for hash := range sender {
			if err := HashWrite(s.rw, hash.h); err != nil {
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
	// run stage 2
	if err := util.Sel(ctx, stage2); err != nil {
		return err
	}

	return nil
}
