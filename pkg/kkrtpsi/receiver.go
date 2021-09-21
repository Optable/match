package kkrtpsi

import (
	"context"
	"encoding/binary"
	"encoding/gob"
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
func (r *Receiver) _Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error) {
	// start timer:
	start := time.Now()
	var seeds [cuckoo.Nhash][]byte
	var intersection [][]byte
	var oprfOutput [][]byte
	var oprfOutputSize int
	var cuckooHashTable *cuckoo.Cuckoo
	var input = make(chan [][]byte, 1)
	//var errBus = make(chan error)

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
		go func() {
			// fetch local ID and insert
			for identifier := range identifiers {
				cuckooHashTable.Insert(identifier)
			}

			var oprfInputs = make([][]byte, cuckooHashTable.BucketSize())
			var i = 0
			for in := range cuckooHashTable.OPRFInput() {
				oprfInputs[i] = in
				i++
			}
			input <- oprfInputs
			close(input)
		}()

		// send size
		if err := binary.Write(r.rw, binary.BigEndian, &n); err != nil {
			return err
		}

		// end stage1
		end1 := time.Now()
		fmt.Println("Stage1: ", end1.Sub(start))
		return nil
	}

	// stage 2: prepare OPRF receive input and run Receive to get OPRF output
	stage2 := func() error {
		oprfInputSize := int64(cuckooHashTable.Len())
		oprfOutputSize = findK(oprfInputSize)

		// inform the sender of the size
		// its about to receive
		if err := binary.Write(r.rw, binary.BigEndian, &oprfInputSize); err != nil {
			return err
		}

		oReceiver, err := oprf.NewKKRT(int(oprfInputSize), oprfOutputSize, ot.Simplest, false)
		if err != nil {
			return err
		}

		oprfOutput, err = oReceiver.Receive(<-input, r.rw)
		if err != nil {
			return err
		}

		// sanity check
		if len(oprfOutput) != int(oprfInputSize) {
			return fmt.Errorf("received number of OPRF outputs should be the same as cuckoohash bucket size")
		}

		// end stage2
		end2 := time.Now()
		fmt.Println("Stage2: ", end2.Sub(start))
		return nil
	}

	// stage 3: read remote encoded identifiers and compare
	//          to produce intersections
	stage3 := func() error {
		bucket := cuckooHashTable.Bucket()
		localChan := make(chan uint64, len(oprfOutput))
		// hash local oprf output
		go func() {
			hasher, _ := hash.New(hash.Highway, seeds[0])
			for _, output := range oprfOutput {
				localChan <- hasher.Hash64(output)
			}

			close(localChan)
		}()

		// read number of remote IDs
		var remoteN int64
		if err := binary.Read(r.rw, binary.BigEndian, &remoteN); err != nil {
			return err
		}

		// read cuckoo.Nhash number of hastable table of encoded remote IDs
		var remoteEncodings [cuckoo.Nhash]map[uint64]bool
		decoder := gob.NewDecoder(r.rw)
		for i := range remoteEncodings {
			// read encoded id and insert
			remoteEncodings[i] = make(map[uint64]bool, remoteN)
			if err := decoder.Decode(&remoteEncodings[i]); err != nil {
				return err
			}
		}

		// intersect
		// intersect
		var hIdx uint8
		var localOutput uint64
		for value := range bucket {
			localOutput = <-localChan
			// compare oprf output to every encoded in remoteHashTable at hIdx
			if !value.Empty() {
				hIdx = value.GetHashIdx()
				if remoteEncodings[hIdx][localOutput] {
					intersection = append(intersection, value.GetItem())
					// dedup
					delete(remoteEncodings[hIdx], localOutput)
				}
			}
		}

		// end stage3
		end3 := time.Now()
		fmt.Println("stage3: ", end3.Sub(start))
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
