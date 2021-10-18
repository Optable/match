package kkrtpsi

import (
	"bufio"
	"context"
	"crypto/rand"
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
	// start timer:
	start := time.Now()
	timer := time.Now()
	var mem uint64

	var seeds [cuckoo.Nhash][]byte
	var remoteN int64       // receiver size
	var oprfInputSize int64 // nb of OPRF keys

	var oprfKeys oprf.Key
	var hashedIds = make(chan hashable, n)

	// stage 1: sample 3 hash seeds and write them to receiver
	// for cuckoo hashing parameters agreement.
	// read local ids and store the potential bucket indexes for each id.
	stage1 := func() error {
		// sample Nhash hash seeds
		for i := range seeds {
			seeds[i] = make([]byte, hash.SaltLength)
			// slower read, but we need robust pseudorandomness for the hash seeds.
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

		// exhaust local ids, and precompute all potential
		// hashes and store them using the same
		// cuckoo hash table parameters as the receiver.
		go func() {
			cuckooHashTable := cuckoo.NewDummyCuckoo(uint64(remoteN), seeds)
			for id := range identifiers {
				hashedIds <- hashable{identifier: id, bucketIdx: cuckooHashTable.BucketIndices(id)}
			}
			// no longer need it.
			cuckooHashTable = nil
			close(hashedIds)
		}()

		// end stage1
		timer, mem = printStageStats("Stage 1", start, start, 0)
		return nil
	}

	// stage 2: act as sender in OPRF, and receive OPRF keys
	stage2 := func() error {
		// receive the number of OPRF
		if err := binary.Read(s.rw, binary.BigEndian, &oprfInputSize); err != nil {
			return err
		}

		// instantiate OPRF sender with agreed parameters
		oSender, err := oprf.NewOPRF(oprf.ImprvKKRT, int(oprfInputSize), ot.NaorPinkas)
		if err != nil {
			return err
		}

		oprfKeys, err = oSender.Send(s.rw)
		if err != nil {
			return err
		}

		// end stage2
		timer, mem = printStageStats("Stage 2", timer, start, mem)
		return nil
	}

	// stage 3: compute all possible OPRF output using keys obtained from stage2
	stage3 := func() error {
		// inform the receiver the number of local ID
		if err := binary.Write(s.rw, binary.BigEndian, &n); err != nil {
			return err
		}

		hasher, err := hash.New(hash.Highway, seeds[0])
		if err != nil {
			return err
		}

		localEncodings := EncodeAndHashAllParallel(oprfKeys, hasher, hashedIds)

		// Add a buffer of 64k to amortize syscalls cost
		var bufferedWriter = bufio.NewWriterSize(s.rw, 1024*64)
		defer bufferedWriter.Flush()

		for hashedEncodings := range localEncodings {
			// send all 3 encoding at once
			if err := EncodesWrite(bufferedWriter, hashedEncodings); err != nil {
				return fmt.Errorf("stage3: %v", err)
			}
		}

		// end stage3
		_, _ = printStageStats("Stage 3", timer, start, mem)
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
