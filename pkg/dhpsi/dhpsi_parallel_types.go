package dhpsi

import (
	"runtime"
)

const (
	batchSize = 8
)

var (
	parallelism = runtime.NumCPU()
	dmBus       chan dmOp
	mBus        chan mOp
)

type dmOp struct {
	gr Ristretto
	b  dmBatch
	// closure
	f func(dmBatch)
}

type dmBatch struct {
	// start sequence of this batch
	seq int64
	// size of this batch
	s int64
	// buffer indentifiers in
	batch [][]byte
	// buffer points out
	points [][EncodedLen]byte
}

type mOp struct {
	gr Ristretto
	b  mBatch
	// closure
	f func(mBatch)
}

type mBatch struct {
	// batch number
	n int
	// size of this batch
	s int64
	// buffer points in
	batch [][EncodedLen]byte
	// buffer points out
	points [][EncodedLen]byte
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
				op.gr.DeriveMultiply(&b.points[k], v)
			}
			// closure
			op.f(b)
		case op := <-m:
			// extract the batch
			b := op.b
			// multiply the points into points
			for k, v := range b.batch {
				op.gr.Multiply(&b.points[k], v)
			}
			// closure
			op.f(b)
		}
	}
}

// make a new batch for the DM operation
func makeDMBatch(seq, batchSize int64) dmBatch {
	return dmBatch{seq: seq, s: batchSize, batch: make([][]byte, batchSize), points: make([][EncodedLen]byte, batchSize)}
}

// make a new mBatch of exacly the right size needed
// so that readers do not block or return EOF
func makeMBatch(n int, batchSize int64) mBatch {
	return mBatch{n: n, s: batchSize, batch: make([][EncodedLen]byte, batchSize), points: make([][EncodedLen]byte, batchSize)}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
