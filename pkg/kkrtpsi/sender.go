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
	hashes     [cuckoo.Nhash]uint64
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
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) error {
	var seeds [cuckoo.Nhash][]byte
	var oSender oprf.OPRF
	var oprfKeys []oprf.Key
	var remoteN int
	// keys is h_1(x), value is [2]uint64, that stores the rest of the hashed values
	hashedIds := make([]hashable, n)

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

		// read all local ids, and store them as hashbles
		c := cuckoo.NewCuckoo(uint64(n), seeds)
		var i = 0
		for id := range identifiers {
			hashedIds[i] = hashable{identifier: id, hashes: c.Hash(id)}
			i++
		}
		return nil
	}

	// stage 2:
	stage2 := func() error {
		// receive the number of OPRF
		var inputLen int64
		if err := binary.Read(s.rw, binary.BigEndian, &inputLen); err != nil {
			return err
		}

		oSender, err := oprf.NewKKRT(int(inputLen), findK(inputLen), ot.Simplest, false)
		if err != nil {
			return err
		}

		oprfKeys, err = oSender.Send(s.rw)
		if err != nil {
			return err
		}
		return nil
	}

	// stage 3: compute all possible OPRF output using keys obtained from stage2
	stage3 := func() error {
		/*
			// in cuckoo hash table
			var h [][]byte
			// in cuckoo stash
				s := make([][]byte, len(oprfKeys)-cuckoo.Factor*remoteN)

				// encode (need to fix this)
				for i, hashable := range hashedIds {
					for _, hash := range hashable.hashes {
						encoded, err := oSender.Encode(oprfKeys[hash], hashable.identifier)
						if err != nil {
							return err
						}

						h = append(h, encoded)
					}
				}
		*/
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

	fmt.Println(oSender)
	fmt.Println(oprfKeys[:2])
	return nil
}
