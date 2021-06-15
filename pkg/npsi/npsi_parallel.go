package npsi

import (
	"runtime"
	"sync"

	"github.com/optable/match/internal/hash"
)

//
// parallel hashing engine
//

const (
	batchSize = 512
)

var hOpBus = make(chan hOp)

// hOp is a hash operation
// being sent to the hashing engine
type hOp struct {
	hh hash.Hasher
	l  int
	x  [][]byte
	h  []uint64
	f  func(h hOp)
}

func handler() {
	for {
		select {
		case op := <-hOpBus:
			var h = make([]uint64, op.l)
			for i := 0; i < op.l; i++ {
				h[i] = op.hh.Hash64(op.x[i])
			}
			op.h = h
			op.f(op)
		}
	}
}

// initHasher to start a bunch of hash handlers
func init() {
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go handler()
	}
}

// HashAllParallel reads all identifiers from identifiers
// and parallel hashes them until identifiers closes
func HashAllParallel(h hash.Hasher, identifiers <-chan []byte) <-chan hashPair {
	// one wg.Add() per batch + one for the batcher go routine
	var wg sync.WaitGroup
	var pairs = make(chan hashPair)

	f := func(op hOp) {
		// pump everything out
		for i := 0; i < op.l; i++ {
			pairs <- hashPair{x: op.x[i], h: op.h[i]}
		}
		wg.Done()
	}

	// parallel hash is overkill here probably.
	// these hash operations are super fast.
	// we do get localized pseudo randomness out of this
	// since no effort is made to re-order finished batches
	// batchSize has to be big enought to amortize the cost of the
	// heavy machinery deployed here
	wg.Add(1)
	go func() {
		defer wg.Done()
		var i = 0
		// init a first batch
		var batch = makeOp(h, batchSize, f)
		for identifier := range identifiers {
			// accumulate a batch
			batch.x[i] = identifier
			i++
			// send it out?
			if i == batchSize {
				wg.Add(1)
				hOpBus <- batch
				// reset batch
				batch = makeOp(h, batchSize, f)
				i = 0
			}
		}
		// anything left here?
		if i != 0 {
			batch.l = i
			wg.Add(1)
			hOpBus <- batch
		}
	}()

	// turn the lights off on your way out
	// it has to happen after at least once batch
	// has been sent for processing
	go func() {
		wg.Wait()
		close(pairs)
	}()

	return pairs
}

func makeOp(hh hash.Hasher, l int, f func(hOp)) hOp {
	return hOp{hh: hh, l: l, x: make([][]byte, l), f: f}
}
