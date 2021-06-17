package npsi

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"sync"

	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/util"
)

// stage 1: P2 samples a random salt K and sends it to P1.
// stage 2: P2 receives hashes from P1 and computes the intersection with its own hashes

// Receiver side of the NPSI protocol
type Receiver struct {
	rw io.ReadWriter
}

// NewReceiver returns a receiver initialized to
// use rw as the communication layer
func NewReceiver(rw io.ReadWriter) *Receiver {
	switch v := rw.(type) {
	// disable nagle
	case *net.TCPConn:
		v.SetNoDelay(false)
	}
	return &Receiver{rw: rw}
}

// Intersect on matchables read from the identifiers channel,
// returning the matching intersection, using the NPSI protocol.
// The format of an indentifier is
//  string
func (r *Receiver) Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error) {
	var intersected [][]byte
	var k = make([]byte, hash.SaltLength)

	// stage 1: P2 samples a random salt K and sends it to P1.
	stage1 := func() error {
		// stage1.1: generate a SaltLength salt
		if _, err := rand.Read(k); err != nil {
			return err
		}
		// stage1.2: send k to the sender
		if _, err := r.rw.Write(k); err != nil {
			return err
		}

		return nil
	}

	// stage 2: P2 receives hashes from P1 and computes the intersection with its own hashes
	stage2v2 := func() error {
		//
		var localIDs = make(map[uint64][]byte)
		var remoteIDs = make(map[uint64]bool)
		// get a hasher
		h, err := hash.New(hash.Highway, k)
		if err != nil {
			return err
		}
		// sender sends the number
		// of items its about to write first
		var n int64
		if err := binary.Read(r.rw, binary.BigEndian, &n); err != nil {
			return err
		}
		//
		// stage2 : P2 receives hashes from P1 (Hi) and computes its own hashes from Xj,
		// then the intersection with its own hashes (Hj)
		//
		// make a channel to receive hashes from the sender
		sender := ReadAll(r.rw, n)
		// make a channel to receive local x,h pairs
		receiver := HashAllParallel(h, identifiers)
		// try to intersect and throw out intersected hashes as we get them
		var wg sync.WaitGroup
		// intersect
		wg.Add(2)
		go func() {
			// index the sender
			defer wg.Done()
			for h := range sender {
				remoteIDs[h] = true
			}
		}()
		go func() {
			// index the receiver
			defer wg.Done()
			for pair := range receiver {
				localIDs[pair.h] = pair.x
			}
		}()
		// let the indexing finish
		wg.Wait()
		// intersect
		for h, x := range localIDs {
			if remoteIDs[h] {
				intersected = append(intersected, x)
			}
		}

		// break out
		return nil
	}

	// run stage 1
	if err := util.Sel(ctx, stage1); err != nil {
		return nil, err
	}

	// run stage 2
	if err := util.Sel(ctx, stage2v2); err != nil {
		return intersected, err
	}

	// all went well
	return intersected, nil
}
