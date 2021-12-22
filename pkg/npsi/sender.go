package npsi

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/go-logr/logr"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/util"
)

// stage 1: receive a random salt K from P1
// stage 2: send hashes salted with K to P1

// Sender represents sender side of the NPSI protocol
type Sender struct {
	rw *bufio.ReadWriter
}

// NewSender returns a sender initialized to
// use rw as a buffered communication layer
func NewSender(rw io.ReadWriter) *Sender {
	return &Sender{rw: bufio.NewReadWriter(bufio.NewReader(rw), bufio.NewWriter(rw))}
}

// Send initiates a NPSI exchange
// that are read from identifiers, until identifiers closes.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) error {
	// fetch and set up logger
	logger := logr.FromContextOrDiscard(ctx)
	logger = logger.WithValues("protocol", "npsi")

	// hold k
	var k = make([]byte, hash.SaltLength)
	// stage 1: receive a random salt K from P1
	stage1 := func() error {
		logger.V(1).Info("Starting stage 1")
		if n, err := s.rw.Read(k); err != nil {
			return fmt.Errorf("stage1: %v", err)
		} else if n != hash.SaltLength {
			return hash.ErrSaltLengthMismatch
		}

		logger.V(1).Info("Finished stage 1")
		return nil
	}

	// stage 2: send hashes salted with K to P1
	stage2 := func() error {
		logger.V(1).Info("Starting stage 2")
		// get a hasher
		h, err := hash.NewMetroHasher(k)
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
		s.rw.Flush()

		logger.V(1).Info("Finished stage 2")
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

	logger.V(1).Info("sender finished")
	return nil
}
