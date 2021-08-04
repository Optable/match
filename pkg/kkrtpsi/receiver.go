package kkrtpsi

import (
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
	var cuckooHashTable *cuckoo.Cuckoo

	// stage 1: read the hash seeds from the remote side
	//          initiate a cuckoo hash table and insert all local
	//          IDs into the cuckoo hash table.
	stage1 := func() error {
		var seeds [cuckoo.Nhash][]byte
		for i := range seeds {
			seeds[i] = make([]byte, hash.SaltLength)
			if n, err := r.rw.Read(seeds[i]); err != nil {
				return fmt.Errorf("stage1: %v", err)
			} else if n != hash.SaltLength {
				return hash.ErrSaltLengthMismatch
			}
		}

		// instantiate cuckoo hash table
		cuckooHashTable = cuckoo.NewCuckoo(uint64(n), seeds)
		// fetch local ID and insert
		for identifier := range identifiers {
			if err := cuckooHashTable.Insert(identifier); err != nil {
				return err
			}
		}
		return nil
	}

	// stage 2: prepare OPRF receive input and run Receive to get OPRF output
	stage2 := func() error {
		input := cuckooHashTable.OPRFInput()
		inputLen := len(input)

		// inform the sender of the size
		// its about to receive
		if err := binary.Write(r.rw, binary.BigEndian, &inputLen); err != nil {
			return err
		}

		oReceiver, err := oprf.NewKKRT(inputLen, findK(inputLen), ot.Simplest, false)
		if err != nil {
			return err
		}

		oprfOutput, err = oReceiver.Receive(input, r.rw)
		if err != nil {
			return err
		}
		return nil
	}

	// stage 3: read local IDs and compare with the remote bloomfilter
	//          to produce intersections
	stage3 := func() error {
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

	fmt.Println(oprfOutput[:2])
	return intersected, nil
}
