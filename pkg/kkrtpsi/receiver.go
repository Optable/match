package kkrtpsi

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/oprf"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
)

// stage 1: read the 3 hash seeds for cuckoo hash, read local IDs until exhaustion
//          and insert them all into a cuckoo hash table
// stage 2: OPRF Receive
// stage 3: receive remote OPRF outputs and intersect

// Receiver side of the KKRTPSI protocol
type Receiver struct {
	rw io.ReadWriter
}

// NewReceiver returns a KKRT receiver initialized to
// use rw as the communication layer
func NewReceiver(rw io.ReadWriter) *Receiver {
	return &Receiver{rw: rw}
}

// Intersect on matchables read from the identifiers channel,
// returning the matching intersection, using the KKRTPSI protocol.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (r *Receiver) Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error) {
	// start timer:
	start := time.Now()
	timer := time.Now()
	var mem uint64

	var seeds [cuckoo.Nhash][]byte
	var intersection [][]byte
	var oprfOutput [][]byte
	var cuckooHashTable *cuckoo.Cuckoo

	// stage 1: read the hash seeds from the remote side
	//          initiate a cuckoo hash table and insert all local
	//          IDs into the cuckoo hash table.
	stage1 := func() error {
		for i := range seeds {
			seeds[i] = make([]byte, hash.SaltLength)
			if _, err := io.ReadFull(r.rw, seeds[i]); err != nil {
				return fmt.Errorf("stage1: %v", err)
			}
		}

		// instantiate cuckoo hash table
		cuckooHashTable = cuckoo.NewCuckoo(uint64(n), seeds)
		err := cuckooHashTable.Insert(identifiers)
		if err != nil {
			return err
		}

		// send size
		if err := binary.Write(r.rw, binary.BigEndian, &n); err != nil {
			return err
		}

		// end stage1
		timer, mem = printStageStats("Stage 1", start, start, 0)
		return nil
	}

	// stage 2: prepare OPRF receive input and run Receive to get OPRF output
	stage2 := func() error {
		oprfInputSize := int64(cuckooHashTable.Len())

		// inform the sender of the size
		// its about to receive
		if err := binary.Write(r.rw, binary.BigEndian, oprfInputSize); err != nil {
			return err
		}

		oReceiver, err := oprf.NewOPRF(oprf.ImprvKKRT, int(oprfInputSize), ot.NaorPinkas)
		if err != nil {
			return err
		}

		oprfOutput, err = oReceiver.Receive(cuckooHashTable, r.rw)
		if err != nil {
			return err
		}

		// sanity check
		if len(oprfOutput) != int(oprfInputSize) {
			return fmt.Errorf("received number of OPRF outputs should be the same as cuckoo hash bucket size")
		}

		// end stage2
		timer, mem = printStageStats("Stage 2", timer, start, mem)
		return nil
	}

	// stage 3: read remote encoded identifiers and compare
	//          to produce intersections
	stage3 := func() error {

		// Hash and index all local encodings
		// the hash value of the oprf encoding is the key
		// the index of the corresponding ID is the value
		var localEncodings [cuckoo.Nhash]map[uint64]uint64
		for i := range localEncodings {
			localEncodings[i] = make(map[uint64]uint64, n)
		}
		// hash local oprf output
		hasher, _ := hash.New(hash.Highway, seeds[0])
		for bIdx := uint64(0); bIdx < cuckooHashTable.Len(); bIdx++ {
			// check if it was an empty input
			if idx, _ := cuckooHashTable.GetBucket(bIdx); idx != 0 {
				// insert into proper map
				hIdx, _ := cuckooHashTable.Exists(idx)
				localEncodings[hIdx][hasher.Hash64(oprfOutput[bIdx])] = idx
			}
		}

		// read number of remote IDs
		var remoteN int64
		if err := binary.Read(r.rw, binary.BigEndian, &remoteN); err != nil {
			return err
		}

		// Add a buffer of 64k to amortize syscalls cost
		var bufferedReader = bufio.NewReaderSize(r.rw, 1024*64)

		// read remote encodings and intersect
		var remoteEncodings [cuckoo.Nhash]uint64
		for i := int64(0); i < remoteN; i++ {
			// read 3 possible encodings
			if err := EncodesRead(bufferedReader, &remoteEncodings); err != nil {
				return err
			}

			// intersect
			for hashIdx, remoteHash := range remoteEncodings {
				if idx, ok := localEncodings[hashIdx][remoteHash]; ok {
					id, err := cuckooHashTable.GetItem(idx)
					if err != nil {
						return err
					}
					intersection = append(intersection, id)
				}
			}
		}
		// end stage3
		_, _ = printStageStats("Stage 3", timer, start, mem)
		return nil
	}

	// run stage1
	if err := util.Sel(ctx, stage1); err != nil {
		return intersection, err
	}

	// run stage2
	if err := util.Sel(ctx, stage2); err != nil {
		return intersection, err
	}

	// run stage3
	if err := util.Sel(ctx, stage3); err != nil {
		return intersection, err
	}

	return intersection, nil
}
