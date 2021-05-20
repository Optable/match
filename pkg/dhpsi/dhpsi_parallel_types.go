package dhpsi

import (
	"runtime"
)

const (
	batchSize = 128
)

var (
	parallelism = runtime.NumCPU()
	dmBus       chan dmOp
	mBus        chan mOp
)

type dmOp struct {
	r Ristretto
	b dmBatch
	// closure
	f func(dmBatch)
}

type dmBatch struct {
	// start sequence of this batch
	seq int64
	// size of this batch
	s int64
	// buffer in
	batch [][]byte
	// points out
	points [][EncodedLen]byte
}

type mBatch struct {
	// start sequence of this batch
	seq int64
	// size of this batch
	s int64
	// points in
	batch [][EncodedLen]byte
	// points out
	points [][EncodedLen]byte
}

type mOp struct {
	r Ristretto
	b mBatch
	// closure
	f func(mBatch)
}

func init() {
	bus1, bus2 := make(chan dmOp), make(chan mOp)
	dmBus, mBus = bus1, bus2

	for i := 0; i < parallelism; i++ {
		go parallelH(bus1, bus2)
	}
}

func parallelH(dm chan dmOp, m chan mOp) {
	for {
		select {
		case op := <-dm:
			// extract the batch
			b := op.b
			// derive/multiply the identifiers into points
			for k, v := range b.batch {
				b.points[k] = op.r.DeriveMultiply(v)
			}
			// closure
			op.f(b)
		case op := <-m:
			// extract the batch
			b := op.b
			// multiply the points into points
			for k, v := range b.batch {
				b.points[k] = op.r.Multiply(v)
			}
			// closure
			op.f(b)
		}
	}
}

func makeDMBatch(max, seq, batchSize int64) (dmBatch, bool) {
	// next batch size is min (batchSize, max-seq)
	s := min(batchSize, max-seq)
	if s == 0 {
		return dmBatch{}, false
	} else {
		return dmBatch{seq: seq, s: s, batch: make([][]byte, s), points: make([][EncodedLen]byte, s)}, true
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
