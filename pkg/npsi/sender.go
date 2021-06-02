package npsi

import (
	"context"
	"io"
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
	// stage 1: receive a random salt K from P1
	var k = make([]byte, SaltLength)
	if _, err := s.rw.Read(k); err != nil {
		return err
	} else {

	}

}
