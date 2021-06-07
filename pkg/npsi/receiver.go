package npsi

import (
	"context"
	"crypto/rand"
	"io"
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
	return &Receiver{rw: rw}
}

// Intersect on matchables read from the identifiers channel,
// returning the matching intersection, using the NPSI protocol.
// The format of an indentifier is
//  PREFIX:MATCHABLE
func (r *Receiver) Intersect(ctx context.Context, identifiers <-chan []byte) ([][]byte, error) {
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
	stage2 := func() error {
		//
		var localIDs = make(map[uint64][]byte)
		var remoteIDs = make(map[uint64]bool)
		// get a hasher
		h, err := hash.New(hash.SIP, k)
		if err != nil {
			return err
		}

		//
		// stage2 : P2 receives hashes from P1 (Hi) and computes its own hashes from Xj,
		// then the intersection with its own hashes (Hj)
		//
		// make a channel to receive hashes from the sender
		sender := ReadAll(r.rw)
		// make a channel to receive local x,h pairs
		receiver := HashAll(h, identifiers)
		// try to intersect and throw out intersected hashes as we get them,
		// when the sender and the receiver are exhausted, intersect the rest
		var c1 = make(chan uint64)
		var c2 = make(chan hashPair)
		var done = make(chan bool)
		var wg sync.WaitGroup

		wg.Add(2)
		// drain the sender
		go func() {
			defer wg.Done()
			for Hi := range sender {
				c1 <- Hi
			}
		}()
		// drain the receiver
		go func() {
			defer wg.Done()
			for pair := range receiver {
				c2 <- pair
			}
		}()
		// intersect
		go func() {
			for {
				select {
				case Hi := <-c1:
					// do we have an intersection?
					if identifier, ok := localIDs[Hi]; ok {
						// we do
						intersected = append(intersected, identifier)
						// prune it
						delete(localIDs, Hi)
					} else {
						// we dont, cache this
						remoteIDs[Hi] = true
					}

				case pair := <-c2:
					// do we have an intersection?
					if remoteIDs[pair.h] {
						// we do
						intersected = append(intersected, pair.x)
						// prune it
						delete(remoteIDs, pair.h)
					} else {
						// we dont, cache this
						localIDs[pair.h] = pair.x
					}

				case <-done:
					return
				}
			}
		}()
		// let the intersection finish
		wg.Wait()
		// kill the intersection goroutine
		close(done)
		// break out
		return nil
	}

	// run stage 1
	if err := util.Sel(ctx, stage1); err != nil {
		return nil, err
	}

	// run stage 2
	if err := util.Sel(ctx, stage2); err != nil {
		return intersected, err
	}

	// all went well
	return intersected, nil
}
