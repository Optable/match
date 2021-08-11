package kkrtpsi

import (
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
	var intersected [][]byte
	var oprfOutput [][]byte
	var oprfOutputSize int
	var cuckooHashTable *cuckoo.Cuckoo

	// stage 1: read the hash seeds from the remote side
	//          initiate a cuckoo hash table and insert all local
	//          IDs into the cuckoo hash table.
	stage1 := func() error {
		var seeds [cuckoo.Nhash][]byte
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
		// fetch local ID and insert
		for identifier := range identifiers {
			if err := cuckooHashTable.Insert(identifier); err != nil {
				return err
			}
		}

		// end stage1
		end1 := time.Now()
		fmt.Println("Stage1: ", end1.Sub(start))
		return nil
	}

	// stage 2: prepare OPRF receive input and run Receive to get OPRF output
	stage2 := func() error {
		input := cuckooHashTable.OPRFInput()
		oprfInputSize := int64(len(input))
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

		oprfOutput, err = oReceiver.Receive(input, r.rw)
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
		// read number of remote IDs
		var remoteN int64
		if err := binary.Read(r.rw, binary.BigEndian, &remoteN); err != nil {
			return err
		}

		// read cuckoo.Nhash number of hastable table of encoded remote IDs
		var remoteHashtables = make([]map[string]bool, cuckoo.Nhash)
		var remoteStashes = make([]map[string]bool, cuckooHashTable.StashSize())
		encoded := make([]byte, oprfOutputSize)

		for i := range remoteHashtables {
			// read encoded id and insert
			remoteHashtables[i] = make(map[string]bool)
			for j := int64(0); j < remoteN; j++ {
				if _, err := io.ReadFull(r.rw, encoded); err != nil {
					return fmt.Errorf("stage3: %v", err)
				}

				remoteHashtables[i][string(encoded)] = true
			}
		}

		// read stashSize number of stash of encoded remote IDs
		for i := range remoteStashes {
			remoteStashes[i] = make(map[string]bool, remoteN)
			// read encoded id and insert to map
			for j := int64(0); j < remoteN; j++ {
				if _, err := io.ReadFull(r.rw, encoded); err != nil {
					return fmt.Errorf("stage3: %v", err)
				}

				remoteStashes[i][string(encoded)] = true
			}
		}

		// intersect
		localStash := cuckooHashTable.Stash()
		localBucket := cuckooHashTable.Bucket()
		bucketSize := cuckooHashTable.BucketSize()
		for i, v := range localStash {
			// compare oprf output to every encoded in remoteStashes at index i
			if remoteStashes[i][string(oprfOutput[i+bucketSize])] {
				intersected = append(intersected, v.GetItem())
				// dedup
				// how?
			}
		}

		for k, v := range localBucket {
			// compare oprf output to every encoded in remoteHashTable at hIdx
			hIdx := localBucket[k].GetHashIdx()
			if remoteHashtables[hIdx][string(oprfOutput[localBucket[k].GetBucketIdx()])] {
				intersected = append(intersected, v.GetItem())
				// dedup
				delete(localBucket, k)
			}
		}

		// end stage3
		end3 := time.Now()
		fmt.Println("stage3: ", end3.Sub(start))
		return nil
	}

	// run stage1
	if err := util.Sel(ctx, stage1); err != nil {
		return intersected, err
	}

	// run stage2
	if err := util.Sel(ctx, stage2); err != nil {
		return intersected, err
	}

	// run stage3
	if err := util.Sel(ctx, stage3); err != nil {
		return intersected, err
	}

	return intersected, nil
}
