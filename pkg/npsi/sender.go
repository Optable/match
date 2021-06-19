package npsi

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/util"
)

// stage 1: receive a random salt K from P1
// stage 2: send hashes salted with K to P1

// Sender side of the NPSI protocol
type Sender struct {
	rw io.ReadWriter
}

// NewSender returns a sender initialized to
// use rw as the communication layer
func NewSender(rw io.ReadWriter) *Sender {
	switch v := rw.(type) {
	// enable nagle
	case *net.TCPConn:
		v.SetNoDelay(false)
	}
	return &Sender{rw: rw}
}

// Send initiates a NPSI exchange
// that are read from identifiers, until identifiers closes.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) error {
	// hold k
	var k = make([]byte, hash.SaltLength)
	// stage 1: receive a random salt K from P1
	stage1 := func() error {
		if n, err := s.rw.Read(k); err != nil {
			return fmt.Errorf("stage1: %v", err)
		} else if n != hash.SaltLength {
			return hash.ErrSaltLengthMismatch
		}
		return nil
	}

	// stage 2: send hashes salted with K to P1
	stage2 := func() error {
		// get a hasher
		if h, err := hash.New(hash.Highway, k); err != nil {
			return err
		} else {
			// inform the receiver of the size
			// its about to receive
			if err := binary.Write(s.rw, binary.BigEndian, &n); err != nil {
				return err
			}

			// Add a buffer of 64k to amortize syscalls cost
			bufW := bufio.NewWriterSize(s.rw, 1024*64)
			defer bufW.Flush()

			// make a channel to receive local x,h pairs
			sender := HashAllParallel(h, identifiers)
			// exhaust the hashes into the receiver
			for hash := range sender {
				if err := HashWrite(bufW, hash.h); err != nil {
					return fmt.Errorf("stage2: %v", err)
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
