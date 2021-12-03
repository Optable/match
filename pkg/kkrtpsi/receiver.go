package kkrtpsi

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/go-logr/logr"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/oprf"
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
func (r *Receiver) Intersect(ctx context.Context, n int64, identifiers <-chan []byte) (intersection [][]byte, err error) {
	// fetch and set up logger
	logger := logr.FromContextOrDiscard(ctx)
	logger = logger.WithValues("protocol", "kkrtpsi")

	// start timer:
	start := time.Now()
	timer := time.Now()
	var mem uint64

	var seeds [cuckoo.Nhash][]byte
	var oprfOutput = make([]map[uint64]uint64, cuckoo.Nhash)
	var cuckooHashTable *cuckoo.Cuckoo
	var secretKey []byte

	// stage 1: read the hash seeds from the remote side
	//          initiate a cuckoo hash table and insert all local
	//          IDs into the cuckoo hash table.
	stage1 := func() error {
		logger.V(1).Info("Starting stage 1")
		for i := range seeds {
			seeds[i] = make([]byte, hash.SaltLength)
			if _, err := io.ReadFull(r.rw, seeds[i]); err != nil {
				return fmt.Errorf("stage1: %v", err)
			}
		}

		// send size
		if err := binary.Write(r.rw, binary.BigEndian, &n); err != nil {
			return err
		}

		// instantiate cuckoo hash table
		cuckooHashTable = cuckoo.NewCuckoo(uint64(n), seeds)
		for id := range identifiers {
			if err = cuckooHashTable.Insert(id); err != nil {
				return err
			}
		}

		// receive secret key for AES-128 (16 byte)
		// use the first seed as the 32-byte key for highway hashing
		secretKey = make([]byte, 16)
		if _, err := io.ReadFull(r.rw, secretKey); err != nil {
			return fmt.Errorf("stage1: %v", err)
		}

		// end stage1
		timer, mem = printStageStats(logger, 1, start, start, 0)
		logger.V(1).Info("Finished stage 1")
		return nil
	}

	// stage 2: prepare OPRF receive input and run Receive to get OPRF output
	stage2 := func() error {
		logger.V(1).Info("Starting stage 2")
		oprfInputSize := int(cuckooHashTable.Len())
		oprfOutput, err = oprf.NewOPRF(oprfInputSize).Receive(cuckooHashTable, secretKey, r.rw)
		if err != nil {
			return err
		}

		// end stage2
		timer, mem = printStageStats(logger, 2, timer, start, mem)
		logger.V(1).Info("Finished stage 2")
		return nil
	}

	// stage 3: read remote encoded identifiers and compare
	//          to produce intersections
	stage3 := func() error {
		logger.V(1).Info("Starting stage 3")
		// read number of remote IDs
		var remoteN int64
		if err := binary.Read(r.rw, binary.BigEndian, &remoteN); err != nil {
			return err
		}

		// Add a buffer of 64k to amortize syscalls cost
		var bufferedReader = bufio.NewReaderSize(r.rw, 1024*64)

		// read remote encodings and intersect
		for i := int64(0); i < remoteN; i++ {
			// read 3 possible encodings
			var remoteEncoding [cuckoo.Nhash]uint64
			if err := EncodingsRead(bufferedReader, &remoteEncoding); err != nil {
				return err
			}
			// intersect
			for hashIdx, remoteHash := range remoteEncoding {
				if idx, ok := oprfOutput[hashIdx][remoteHash]; ok {
					id, _ := cuckooHashTable.GetItemWithHash(idx)
					if id == nil {
						return fmt.Errorf("failed to retrieve item #%v", idx)
					}
					intersection = append(intersection, id)
					// dedup
					delete(oprfOutput[hashIdx], remoteHash)
				}
			}
		}
		// end stage3
		_, _ = printStageStats(logger, 3, timer, start, mem)
		logger.V(1).Info("Finished stage 3")
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
