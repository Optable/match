package dhpsi

import (
	"encoding/binary"
	"io"
	"sync"
)

//
// WRITERS
//

// types
type DeriveMultiplyParallelShuffler struct {
	w        io.Writer
	seq, max int64
	r        Ristretto
	// precomputed order to send things in
	permutations []int64
	// pre-processing batch buffer
	b dmBatch
	// last batch sync
	wg sync.WaitGroup
	// post-processing point buffer
	points [][EncodedLen]byte
}

// NewShufflerDirectEncoder returns a dhpsi encoder that hashes, encrypts
// and shuffles matchable values on n sequences of bytes to be sent out.
// It first computes a permutation table and subsequently sends out sequences ordered
// by the precomputed permutation table. This is the first stage of doing a DH exchange.
func NewDeriveMultiplyParallelShuffler(w io.Writer, n int64, r Ristretto) (*DeriveMultiplyParallelShuffler, error) {
	// send the max value first
	if err := binary.Write(w, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	batch, _ := makeDMBatch(n, 0, batchSize)
	// and create the encoder
	enc := &DeriveMultiplyParallelShuffler{w: w, max: n, r: r, permutations: initP(n), b: batch, points: make([][EncodedLen]byte, n)}
	enc.wg.Add(1)
	return enc, nil
}

// Shuffle one prefixed ID. First derive and then multiply by the
// precomputed scaler, written out to the underlying writer while following
// the order of permutations created at NewDeriveMultiplyShuffler.
// Returns ErrUnexpectedEncodeByte when the whole expected sequence has been sent.
func (enc *DeriveMultiplyParallelShuffler) Shuffle(identifier []byte) (err error) {
	// ignore any encode past the max encodes
	// we're configured for
	if enc.seq == enc.max {
		return ErrUnexpectedEncodeByte
	}

	// closure for workers
	f := func(b dmBatch) {
		for k, v := range b.points {
			enc.points[int64(k)+b.seq] = v
		}
		// signal a batch is done
		enc.wg.Done()
	}

	// add to the current batch
	next := enc.seq % batchSize
	enc.b.batch[next] = identifier
	enc.seq++
	// process batch?
	if next == enc.b.s-1 {
		dmBus <- dmOp{r: enc.r, b: enc.b, f: f}
		// there's a edge case here. we processed
		// the last batch already and there is no next
		batch, ok := makeDMBatch(enc.max, enc.seq, batchSize)
		enc.b = batch
		if ok {
			enc.wg.Add(1)
		}
	}

	// after we processed everything flush the buffer
	if enc.seq == enc.max {
		// wait for all batches to finish
		enc.wg.Wait()
		for _, p := range enc.permutations {
			if _, err = enc.w.Write(enc.points[p][:]); err != nil {
				return
			}
		}
	}
	return
}

// Permutations returns the permutation matrix
// that was computed on initialization
func (enc *DeriveMultiplyParallelShuffler) Permutations() []int64 {
	return enc.permutations
}

// InvertedPermutations returns the reverse of the permutation matrix
// that was computed on initialization
func (enc *DeriveMultiplyParallelShuffler) InvertedPermutations() []int64 {
	return invertedPermutations(enc.permutations)
}
