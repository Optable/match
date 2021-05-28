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
	gr       Ristretto
	// precomputed order to send things in
	permutations []int64
	// pre-processing batch buffer
	b dmBatch
	// batch sync
	wg sync.WaitGroup
	// post-processing point buffer
	points [][EncodedLen]byte
}

// NewShufflerDirectEncoder returns a dhpsi encoder that hashes, encrypts
// and shuffles matchable values on n sequences of bytes to be sent out.
// It first computes a permutation table and subsequently sends out sequences ordered
// by the precomputed permutation table. This is the first stage of doing a DH exchange.
func NewDeriveMultiplyParallelShuffler(w io.Writer, n int64, gr Ristretto) (*DeriveMultiplyParallelShuffler, error) {
	// send the max value first
	if err := binary.Write(w, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	// create the first batch
	b := makeDMBatch(0, min(batchSize, n))
	// and create the encoder
	enc := &DeriveMultiplyParallelShuffler{w: w, max: n, gr: gr, permutations: initP(n), b: b, points: make([][EncodedLen]byte, n)}
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
		return ErrUnexpectedPoint
	}

	// closure for workers
	f := func(b dmBatch) {
		for k, v := range b.points {
			enc.points[int64(k)+b.seq] = v
		}
		// signal a batch is done
		enc.wg.Done()
	}

	// next is the offset of the next
	// identifier into the current buffer
	next := enc.seq % batchSize
	enc.b.batch[next] = identifier
	enc.seq++
	// process batch?
	if next == enc.b.s-1 {
		dmBus <- dmOp{gr: enc.gr, b: enc.b, f: f}
		// make a new batch
		// there's a edge case here. we processed
		// the last batch already and there is no next
		s := min(batchSize, enc.max-enc.seq)
		if s != 0 {
			enc.b = makeDMBatch(enc.seq, s)
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

//
// READERS
//

// approach #3
// run a goroutine that always fills
// up the buffers and block on submitting
// the work

type MultiplyParallelReader struct {
	r   *Reader
	seq int64
	bus <-chan [EncodedLen]byte
}

// NewMultiplyParallelReader makes a ristretto multiplier reader that sits on the other end
// of the DeriveMultiplyShuffler or the Writer and reads encoded ristretto hashes and
// multiplies them using gr
func NewMultiplyParallelReader(r io.Reader, gr Ristretto) (*MultiplyParallelReader, error) {
	// setup the underlying reader
	rr, err := NewReader(r)
	if err != nil {
		return nil, err
	}
	// start filling
	c := fill(rr, gr)
	// make a new decoder
	dec := &MultiplyParallelReader{r: rr, bus: c}
	return dec, nil
}

// fill workers with jobs to process
// and block on processing the jobs until
// there is nothing left to read
func fill(r *Reader, gr Ristretto) <-chan [EncodedLen]byte {
	var closed = make(chan bool)
	var batches = make(chan mBatch)
	// calculate the total batch size
	var totalBatches = r.max / batchSize
	if r.max%batchSize != 0 {
		totalBatches++
	}

	// left batches counter
	var lb = make(chan int64)
	go func() {
		// TODO: catch the cancel
		// signal here and close LB,
		// that will return 0 in the batch closure
		// and actually close "closed"
		// as it stands a cancelled context
		// on Read wont stop anything until the sockets are closed
		defer close(lb)
		left := totalBatches - 1
		for {
			select {
			case lb <- left:
				left--
				if left == 0 {
					return
				}
			case <-closed:
				return
			}
		}
	}()

	// closure to process finishes
	// batches while also blocking
	// the worker
	f := func(m mBatch) {
		select {
		case batches <- m:
			// one sent out
		}

		// if this is the last batch,
		// close batches
		left := <-lb
		if left == 0 {
			close(batches)
		}
	}

	// poll r and make batches to process
	// until there's nothing left to read
	go func() {
		totalBatches := int(totalBatches)
		for i := 0; i < totalBatches; i++ {
			b := makeMBatch(i, min(batchSize, r.max-r.seq))
			for j := int64(0); j < b.s; j++ {
				// if there's an error here
				// we can't continue
				// otherwise we'll read exactly
				// r.Max()
				err := r.Read(&b.batch[j])
				if err != nil {
					// cancel everything
					closed <- true
					return
				}
			}
			// this will block if the processing queue
			// is full
			mBus <- mOp{gr: gr, b: b, f: f}
		}
		return
	}()

	// signal downstream errors or EOF
	c := make(chan [EncodedLen]byte)
	// read processed batches
	go func() {
		defer close(c)
		var ring = make(map[int]mBatch, parallelism)
		var sent int
		// process batches until batches closes
		for b := range batches {
			// if this is the current batch, write it out
			if sent == b.n {
				copy_out(b, c, closed)
				sent++
			} else {
				// buffer it
				ring[b.n] = b
			}
			// check if buffered
			if b, ok := ring[sent]; ok {
				// do we have it buffered?
				copy_out(b, c, closed)
				delete(ring, sent)
				sent++
			}
		}

		// flush out the rest of the buffer
		for i := 0; i < len(ring); i++ {
			b := ring[sent+i]
			copy_out(b, c, closed)
		}
	}()

	return c
}

func copy_out(b mBatch, c chan [EncodedLen]byte, done chan bool) (n int64) {
	for _, point := range b.points {
		select {
		case c <- point:
			n++
		case <-done:
			return
		}
	}
	return
}

// Read a point from the underyling reader with ristretto
// and write it into point. Returns io.EOF when
// the sequence has been completely read.
func (dec *MultiplyParallelReader) Read(point *[EncodedLen]byte) (err error) {
	// ignore any read past the max size
	// we're configured for
	if dec.seq == dec.r.max {
		return io.EOF
	}
	// do we have data buffered?

	select {
	case p, open := <-dec.bus:
		if !open {
			return io.ErrUnexpectedEOF
		}
		*point = p
		dec.seq++
	}
	return nil
}

// Max is the expected number of matchable
// this decoder will receive
func (dec *MultiplyParallelReader) Max() int64 {
	return dec.r.max
}
