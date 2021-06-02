package npsi

import (
	"runtime"
)

//
// parallel hashing engine
//

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
func initHasher() {
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go handler()
	}
}
