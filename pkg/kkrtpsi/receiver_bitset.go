package kkrtpsi

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/oprf"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
	"github.com/optable/match/pkg/npsi"
)

// stage 1: read the 3 hash seeds for cuckoo hash, read local IDs until exhaustion
//          and insert them all into a cuckoo hash table
// stage 2: OPRF Receive
// stage 3: receive remote OPRF outputs and intersect

// Intersect on matchables read from the identifiers channel,
// returning the matching intersection, using the KKRTPSI protocol.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (r *Receiver) Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error) {
	// start timer:
	start := time.Now()
	var seeds [cuckoo.Nhash][]byte
	var intersection [][]byte
	var oprfOutput []*bitset.BitSet
	var oprfOutputSize int
	var cuckooHashTable *cuckoo.Cuckoo
	var input = make(chan []*bitset.BitSet, 1)
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

			var oprfInputs []*bitset.BitSet
			for in := range cuckooHashTable.OPRFInput() {
				oprfInputs = append(oprfInputs, util.BytesToBitSet(in))
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
		oprfOutputSize = findBitsetK(oprfInputSize)

		// inform the sender of the size
		// its about to receive
		if err := binary.Write(r.rw, binary.BigEndian, &oprfInputSize); err != nil {
			return err
		}

		oReceiver, err := oprf.NewKKRTBitSet(int(oprfInputSize), oprfOutputSize, ot.Simplest, false)
		if err != nil {
			return err
		}

		oprfOutput, err = oReceiver.Receive(<-input, r.rw)
		if err != nil {
			return err
		}

		// sanity check
		//fmt.Println(len(oprfOutput), oprfOutput[0].Len(), oprfInputSize)
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
			var b []byte
			for _, output := range oprfOutput {
				b, _ = output.MarshalBinary()
				localChan <- hasher.Hash64(b)
			}

			close(localChan)
		}()

		// read number of remote IDs
		var remoteN int64
		if err := binary.Read(r.rw, binary.BigEndian, &remoteN); err != nil {
			return err
		}

		// read cuckoo.Nhash number of slice of encoded remote IDs
		var remoteEncodings [cuckoo.Nhash]map[uint64]struct{}
		var u uint64
		for i := range remoteEncodings {
			// read encoded id and insert
			remoteEncodings[i] = make(map[uint64]struct{}, remoteN)
			for j := int64(0); j < remoteN; j++ {
				if err := npsi.HashRead(r.rw, &u); err != nil {
					return err
				}
				remoteEncodings[i][u] = struct{}{}
			}
		}

		// intersect
		var hIdx uint8
		var localOutput uint64
		var exists bool
		for value := range bucket {
			localOutput = <-localChan
			// compare oprf output to every encoded in remoteHashTable at hIdx
			if !value.Empty() {
				hIdx = value.GetHashIdx()
				if _, exists = remoteEncodings[hIdx][localOutput]; exists {
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
