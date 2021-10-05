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
	var seeds [cuckoo.Nhash][]byte
	var intersection [][]byte
	var oprfOutput [][]byte
	var cuckooHashTable *cuckoo.Cuckoo
	var oprfInputs [][]byte

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
		for identifier := range identifiers {
			cuckooHashTable.Insert(identifier)
		}

		oprfInputs = cuckooHashTable.OPRFInput()

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
		oprfInputSize := int64(len(oprfInputs))

		// inform the sender of the size
		// its about to receive
		if err := binary.Write(r.rw, binary.BigEndian, &oprfInputSize); err != nil {
			return err
		}

		oReceiver, err := oprf.NewOPRF(oprf.KKRT, int(oprfInputSize), ot.NaorPinkas)
		if err != nil {
			return err
		}

		oprfOutput, err = oReceiver.Receive(oprfInputs, r.rw)
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

		// Hash and index all local encodings
		// the hash value of the oprf encoding is the key
		// the corresponding ID is the value
		var localEncodings [cuckoo.Nhash]map[uint64][]byte
		for i := range localEncodings {
			localEncodings[i] = make(map[uint64][]byte)
		}
		// hash local oprf output
		hasher, _ := hash.New(hash.Highway, seeds[0])
		for i, input := range oprfInputs {
			// check if it was a dummy input
			if len(input) != 1 && input[0] != 255 {
				// insert into proper map
				localEncodings[input[len(input)-1]][hasher.Hash64(oprfOutput[i])] = input[:len(input)-1]
			}
		}

		// read number of remote IDs
		var remoteN int64
		if err := binary.Read(r.rw, binary.BigEndian, &remoteN); err != nil {
			return err
		}

		// read remote encodings and intersect
		var remoteEncodings [cuckoo.Nhash]uint64
		for i := int64(0); i < remoteN; i++ {
			// read 3 possible encodings
			if err := EncodesRead(r.rw, &remoteEncodings); err != nil {
				return err
			}

			// intersect
			for hashIdx, remoteHash := range remoteEncodings {
				if id, ok := localEncodings[hashIdx][remoteHash]; ok {
					intersection = append(intersection, id)
					// dedup
					delete(localEncodings[hashIdx], remoteHash)
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
