package kkrtpsi

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/oprf"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
)

// stage 1: samples 3 hash seeds and sends them to receiver for cuckoo hash
// stage 2: act as sender in OPRF, and receive OPRF keys
// stage 3: compute OPRF(k, id) and send them to receiver for intersection.

// Sender side of the KKRTPSI protocol
type Sender struct {
	rw io.ReadWriter
}

// hashable stores the possible bucket
// indexes in the receiver cuckoo hash table
type hashable struct {
	identifier []byte
	bucketIdx  [cuckoo.Nhash]uint64
}

// NewSender returns a KKRTPSI sender initialized to
// use rw as the communication layer
func NewSender(rw io.ReadWriter) *Sender {
	return &Sender{rw: rw}
}

// Send initiates a KKRTPSI exchange
// that reads local IDs from identifiers, until identifiers closes.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) (err error) {
	var stashSize int
	var remoteN int64       // receiver size
	var oprfInputSize int64 // nb of OPRF keys

	var oSender oprf.OPRF
	var oprfKeys []oprf.Key
	var hashedIds = make([]hashable, n)

	// stage 1: sample 3 hash seeds and write them to receiver
	// for cuckoo hashing parameters agreement.
	// read local ids and store the potential bucket indexes for each id.
	stage1 := func() error {
		// init randomness source
		rand.Seed(time.Now().UnixNano())

		// sample Nhash hash seeds
		var seeds [cuckoo.Nhash][]byte
		for i := range seeds {
			seeds[i] = make([]byte, hash.SaltLength)
			rand.Read(seeds[i])
			// write it into rw
			if _, err := s.rw.Write(seeds[i]); err != nil {
				return err
			}
		}

		// read remote input size
		if err := binary.Read(s.rw, binary.BigEndian, &remoteN); err != nil {
			return err
		}

		// read all local ids, and precompute all potential
		// hashes and store them using the same
		// cuckoo hash table parameters as the receiver.
		cuckooHashTable := cuckoo.NewCuckoo(uint64(remoteN), seeds)
		var i = 0
		for id := range identifiers {
			hashedIds[i] = hashable{identifier: id, bucketIdx: cuckooHashTable.BucketIndices(id)}
			i++
		}
		stashSize = cuckooHashTable.StashSize()

		return nil
	}

	// stage 2: act as sender in OPRF, and receive OPRF keys
	stage2 := func() error {
		// receive the number of OPRF
		if err := binary.Read(s.rw, binary.BigEndian, &oprfInputSize); err != nil {
			return err
		}

		// instantiate OPRF sender with agreed parameters
		oSender, err = oprf.NewKKRT(int(oprfInputSize), findK(oprfInputSize), ot.Simplest, false)
		if err != nil {
			return err
		}

		oprfKeys, err = oSender.Send(s.rw)
		if err != nil {
			return err
		}

		return nil
	}

	// stage 3: compute all possible OPRF output using keys obtained from stage2
	stage3 := func() error {
		// inform the receiver the number of local ID
		if err := binary.Write(s.rw, binary.BigEndian, &n); err != nil {
			return err
		}

		var hashtable [cuckoo.Nhash][][]byte
		var stash = make([][][]byte, stashSize)
		var stashStartingIdx = int(oprfInputSize) - stashSize

		for i := range hashtable {
			hashtable[i] = make([][]byte, n)
		}

		for i := range stash {
			stash[i] = make([][]byte, n)
		}

		var wg sync.WaitGroup
		var errBus = make(chan error)

		for idx, hash := range hashedIds {
			wg.Add(1)
			go func(idx int, hash hashable) {
				defer wg.Done()
				// encode identifiers that are potentially stored in receiver's cuckoo hash table
				// in any of the cuckoo.Nhash bukcet index and store it.
				for hIdx, bucketIdx := range hash.bucketIdx {
					encoded, err := oSender.Encode(oprfKeys[bucketIdx], append(hash.identifier, uint8(hIdx)))
					if err != nil {
						errBus <- err
					}

					hashtable[hIdx][idx] = encoded
				}
			}(idx, hash)

			wg.Add(1)
			go func(idx int, hash hashable) {
				defer wg.Done()
				// encode identifier that are potentially stored in receiver's cuckoo stash
				// each identifier can be in any of the stash position
				for i := 0; i < stashSize; i++ {
					encoded, err := oSender.Encode(oprfKeys[stashStartingIdx+i], hash.identifier)
					if err != nil {
						errBus <- err
					}

					stash[i][idx] = encoded
				}
			}(idx, hash)
		}

		// Wait for all encode to complete.
		wg.Wait()
		close(errBus)

		//errors?
		for err := range errBus {
			if err != nil {
				return err
			}
		}

		// write out each of the hashtables
		for hIdx := range hashtable {
			for _, encoded := range hashtable[hIdx] {
				if _, err := s.rw.Write(encoded); err != nil {
					return fmt.Errorf("stage3: %v", err)
				}
			}
		}

		// write out each of the stash
		for si := range stash {
			for _, encoded := range stash[si] {
				if _, err := s.rw.Write(encoded); err != nil {
					return fmt.Errorf("stage3: %v", err)
				}
			}
		}

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
