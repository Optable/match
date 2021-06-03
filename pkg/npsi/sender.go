package npsi

import (
	"context"
	"io"

	"github.com/optable/match/internal/util"
)

// stage 1: receive a random salt K from P1
// stage 2: send hashes salted with K to P1

// Sender side of the NPSI protocol
type Sender struct {
	rw io.ReadWriter
}

// Send initiates a NPSI exchange
// that are read from identifiers, until identifiers closes.
// The format of an indentifier is PREFIX:MATCHABLE
// example:
//  e:0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, identifiers <-chan []byte) error {
	// hold k
	var k = make([]byte, SaltLength)
	// stage 1: receive a random salt K from P1
	stage1 := func() error {
		if n, err := s.rw.Read(k); err != nil {
			return err
		} else if n != SaltLength {
			return ErrSaltLengthMismatch
		}
		return nil
	}

	// stage 2: send hashes salted with K to P1
	stage2 := func() error {
		// get a hasher
		if h, err := NewHasher(HashSIP, k); err != nil {
			return err
		} else {
			// make a channel to receive local x,h pairs
			sender := HashAll(h, identifiers)
			// exhaust the hashes into the receiver
			for hash := range sender {
				if err := HashWrite(s.rw, hash.h); err != nil {
					return err
				}
			}
		}
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