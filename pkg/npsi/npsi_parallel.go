package npsi

import (
	"runtime"
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
	h  Hasher
	in [][]byte
	f  func([]uint64)
}

func handler() {
	for {
		select {
		case op := <-hOpBus:
			var out = make([]uint64, len(op.in))
			for k, v := range op.in {
				out[k] = op.h.Hash64(v)
			}
			op.f(out)
		}
	}
}

// initHasher to start a bunch of hash handlers
func init() {
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go handler()
	}
}

// HashAll reads all identifiers from identifiers
// and parallel hashes them until identifiers closes
func HashAll(h Hasher, identifiers <-chan []byte) <-chan hashPair {
	var pairs = make(chan hashPair)
	// parallel hash is overkill here. these hash operations are
	// super fast. we do get localize pseudo randomness out of this
	// since no effort is made to re-order finished batches
	go func() {
		defer close(pairs)
		for identifier := range identifiers {
			// accumulate a batch

		}
	}()
	return pairs
}
