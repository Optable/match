package kkrtpsi

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"

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

		//fmt.Printf("Stage1: cuckoo size: %d\n", cuckooHashTable.Len())
		return nil
	}

	// stage 2: prepare OPRF receive input and run Receive to get OPRF output
	stage2 := func() error {
		input := cuckooHashTable.OPRFInput()
		oprfInputSize := int64(len(input))
		oprfOutputSize = findK(oprfInputSize)

		//fmt.Printf("oprf input size: %d, ", oprfInputSize)

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

		//fmt.Printf("Stage2: OPRF output size: %d, first output: %v\n", len(oprfOutput), oprfOutput[0])
		//fmt.Printf("Stage2: OPRF output size: %d, first output: %v\n", len(oprfOutput), oprfOutput[1])

		// sanity check
		if len(oprfOutput) != int(oprfInputSize) {
			return fmt.Errorf("received number of OPRF outputs should be the same as cuckoohash bucket size")
		}

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
		var remoteHashtables = make([][][]byte, cuckoo.Nhash)
		var remoteStashes = make([][][]byte, cuckooHashTable.StashSize())
		encoded := make([]byte, oprfOutputSize)

		for i := range remoteHashtables {
			// read encoded id and insert
			remoteHashtables[i] = make([][]byte, remoteN)
			for j := range remoteHashtables[i] {
				remoteHashtables[i][j] = make([]byte, oprfOutputSize)
				if _, err := io.ReadFull(r.rw, encoded); err != nil {
					return fmt.Errorf("stage3: %v", err)
				}
				copy(remoteHashtables[i][j], encoded)
			}
		}

		// read stashSize number of stash of encoded remote IDs
		for i := range remoteStashes {
			remoteStashes[i] = make([][]byte, remoteN)
			// read encoded id and insert to map
			for j := range remoteStashes[i] {
				remoteStashes[i][j] = make([]byte, oprfOutputSize)
				if _, err := io.ReadFull(r.rw, encoded); err != nil {
					return fmt.Errorf("stage3: %v", err)
				}

				copy(remoteStashes[i][j], encoded)
			}
		}

		// intersect
		localStash := cuckooHashTable.Stash()
		localBucket := cuckooHashTable.Bucket()
		bucketSize := cuckooHashTable.BucketSize()
		stashStartIdx := int(bucketSize - cuckooHashTable.StashSize())
		fmt.Printf("bucketSize: %d, stashsize: %d, stashStartIdx: %d\n", bucketSize, len(localStash), stashStartIdx)
		for i, v := range localStash {
			// compare oprf output to every encoded in remoteStash at index i
			for j := range remoteStashes[i] {
				if bytes.Equal(oprfOutput[i+stashStartIdx], remoteStashes[i][j]) {
					intersected = append(intersected, v.GetItem())
				}
			}
		}

		fmt.Println(intersected)

		i := 0
		for _, v := range localBucket {
			// compare oprf output to every encoded in remoteHashTable at hIdx
			for j := range remoteHashtables[v.GetHashIdx()] {
				if bytes.Equal(remoteHashtables[v.GetHashIdx()][j], oprfOutput[i]) {
					intersected = append(intersected, v.GetItem())
				}
			}
			i++
		}
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

	//fmt.Println(oprfOutput[:2])
	return intersected, nil
}
