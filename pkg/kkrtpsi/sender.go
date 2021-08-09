package kkrtpsi

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/oprf"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
)

// stage 1: samples 3 hash seeds and sends them to receiver for cuckoo hash
// stage 2: OPRF Send
// stage 3: read local IDs and compute OPRF(k, id) and send them to receiver.

// Sender side of the KKRTPSI protocol
type Sender struct {
	rw io.ReadWriter
}

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
// that are read from identifiers, until identifiers closes.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) (err error) {
	var seeds [cuckoo.Nhash][]byte
	var stashSize int
	var remoteN int64    // receiver size
	var bucketSize int64 // receiver cuckoo bucket size

	var oSender oprf.OPRF
	var oprfKeys []oprf.Key
	var hashedIds = make([]hashable, n)

	// stage 1: sample 3 hash seeds and write them to receiver
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

		// fmt.Printf("Stage1: remote cuckoo stashSize: %d\n", stashSize)
		return nil
	}

	// stage 2:
	stage2 := func() error {
		// receive the number of OPRF

		if err := binary.Read(s.rw, binary.BigEndian, &bucketSize); err != nil {
			return err
		}

		oSender, err = oprf.NewKKRT(int(bucketSize), findK(bucketSize), ot.Simplest, false)
		if err != nil {
			return err
		}

		oprfKeys, err = oSender.Send(s.rw)
		if err != nil {
			return err
		}

		//fmt.Printf("Stage2: OPRFKeys size: %d, first key: %v\n", len(oprfKeys), oprfKeys[0])
		//fmt.Printf("Stage2: OPRFKeys size: %d, first key: %v\n", len(oprfKeys), oprfKeys[1])
		return nil
	}

	// stage 3: compute all possible OPRF output using keys obtained from stage2
	stage3 := func() error {
		// inform the receiver the number of local ID to compute the
		// the number of hash tables and stash to receive
		if err := binary.Write(s.rw, binary.BigEndian, &n); err != nil {
			return err
		}

		stashStartIdx := int(bucketSize - int64(stashSize))

		var hashtable [cuckoo.Nhash][][]byte
		var stash = make([][][]byte, stashSize)

		for _, hashable := range hashedIds {
			for hIdx, bucketIdx := range hashable.bucketIdx {
				// encode identifiers that are potentially stored in receiver's cuckoo hash table
				// in any of the cuckoo.Nhash bukcet index and store it.
				encoded, err := oSender.Encode(oprfKeys[bucketIdx], hashable.identifier)
				if err != nil {
					return err
				}

				hashtable[hIdx] = append(hashtable[hIdx], encoded)
			}

			// encode identifier that are potentially stored in receiver's cuckoo stash
			// each identifier can be in any of the stash position
			for i := 0; i < stashSize; i++ {
				encoded, err := oSender.Encode(oprfKeys[stashStartIdx+i], hashable.identifier)
				if err != nil {
					return nil
				}

				stash[i] = append(stash[i], encoded)
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
