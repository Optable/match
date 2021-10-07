package dhpsi

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/optable/match/internal/permutations"
)

const (
	// EncodedLen is the length of an encoded ristretto point
	EncodedLen = 32
	// EmailPrefixedLen is the length of a prefixed email identifier
	EmailPrefixedLen = 66
)

var (
	ErrUnexpectedPoint = fmt.Errorf("received a point to encode past the configured encoder size")
)

//
// Writers
//

// DeriveMultiplyShuffler contains the necessary
// machineries to derive identifiers into ristretto point,
// multiply them with secret key and permute them.
type DeriveMultiplyShuffler struct {
	w              io.Writer
	max, seq, sent int64
	gr             Ristretto
	// precomputed order to send things in
	p permutations.Permutations
	// buffered in the order received by Shuffle()
	b map[int64][EncodedLen]byte
}

type Writer struct {
	w        io.Writer
	max, seq int64
}

// NewDeriveMultiplyShuffler returns a dhpsi encoder that hashes, encrypts
// and shuffles matchable values on n sequences of bytes to be sent out.
// It first computes a permutation table and subsequently sends out sequences ordered
// by the precomputed permutation table.
// This is the first stage of doing a DH exchange.
func NewDeriveMultiplyShuffler(w io.Writer, n int64, gr Ristretto) (*DeriveMultiplyShuffler, error) {
	if err := binary.Write(w, binary.BigEndian, &n); err != nil {
		return nil, err
	}
	// create the permutations
	p, _ := permutations.NewKensler(n)
	// and create the buffer map & encoder
	b := make(map[int64][EncodedLen]byte, int(float64(n)*0.75))
	return &DeriveMultiplyShuffler{w: w, max: n, gr: gr, p: p, b: b}, nil
}

// Shuffle shuffles one identifier. First derive and then multiply by the
// precomputed scalar, then write out to the underlying writer while following
// the order of permutations created at NewDeriveMultiplyShuffler.
// Returns ErrUnexpectedPoint when the whole expected sequence has been sent.
func (enc *DeriveMultiplyShuffler) Shuffle(identifier []byte) (err error) {
	// ignore any encode past the max encodes
	// we're configured for
	if enc.seq == enc.max {
		return ErrUnexpectedPoint
	}

	// derive/multiply
	var point [EncodedLen]byte
	enc.gr.DeriveMultiply(&point, identifier)

	// we follow the permutation matrix and send
	// or cache incoming matchables
	//next := enc.permutations[enc.sent]
	next := enc.p.Shuffle(enc.sent)
	if next == enc.seq {
		//  we fall perfectly in sequence, write it out
		_, err = enc.w.Write(point[:])
		enc.sent++
	} else {
		// cache the current sequence
		// in the correct, non permutated order
		enc.b[enc.seq] = point
	}
	enc.seq++
	// after we processed everything we will very probably
	// have cached hashes left to send.
	// flush the buffer, in enc.permutations order
	if enc.seq == enc.max {
		// flush the rest of the sequence
		for i := enc.sent; i < enc.max; i++ {
			b := enc.b[enc.p.Shuffle(i)]
			if _, err = enc.w.Write(b[:]); err != nil {
				return
			}
		}
	}
	return
}

// Permutations returns the permutation matrix
// that was computed on initialization
func (enc *DeriveMultiplyShuffler) Permutations() permutations.Permutations {
	return enc.p
}

// NewWriter creates a writer that first sends out
// the total number of items that will be sent out
func NewWriter(w io.Writer, n int64) (*Writer, error) {
	// send the max value first
	if err := binary.Write(w, binary.BigEndian, &n); err != nil {
		return nil, err
	}
	return &Writer{w: w, max: n}, nil
}

// Write writes out the fixed length point to the underlying writer
// while sequencing. Returns ErrUnexpectedPoint if called past
// the configured encoder size
func (w *Writer) Write(point [EncodedLen]byte) (err error) {
	// ignore any encode past the max encodes
	// we're configured for
	if w.seq == w.max {
		return ErrUnexpectedPoint
	}
	//
	if _, err = w.w.Write(point[:]); err != nil {
		return err
	}
	w.seq++
	//
	return
}

//
// READERS
//

type MultiplyReader struct {
	r  *Reader
	gr Ristretto
}

type Reader struct {
	r        io.Reader
	seq, max int64
}

// NewMultiplyReader makes a ristretto multiplier reader that sits on the other end
// of the DeriveMultiplyShuffler or the Writer and reads encoded ristretto hashes and
// multiplies them using gr.
func NewMultiplyReader(r io.Reader, gr Ristretto) (*MultiplyReader, error) {
	rr, err := NewReader(r)
	if err != nil {
		return nil, err
	}
	return &MultiplyReader{r: rr, gr: gr}, nil
}

// Read reads a point from the underlying reader, multiply it with ristretto
// and write it into point. Returns io.EOF when
// the sequence has been completely read.
func (r *MultiplyReader) Read(point *[EncodedLen]byte) (err error) {
	var b [EncodedLen]byte
	if err := r.r.Read(&b); err != nil {
		return err
	}
	r.gr.Multiply(point, b)
	return nil
}

// Max returns the expected number of matchable
// this decoder will receive
func (r *MultiplyReader) Max() int64 {
	return r.r.max
}

// NewReader makes a simple reader that sits on the other end
// of the DeriveMultiplyShuffler or the Writer and reads encoded ristretto points
func NewReader(r io.Reader) (*Reader, error) {
	var max int64
	// extract the max value
	if err := binary.Read(r, binary.BigEndian, &max); err != nil {
		return nil, err
	}
	return &Reader{r: r, max: max}, nil
}

// Read reads a point from the underlying reader and
// write it into p. Returns io.EOF when
// the sequence has been completely read.
func (r *Reader) Read(point *[EncodedLen]byte) (err error) {
	// ignore any read past the max size
	// we're configured for
	if r.seq == r.max {
		return io.EOF
	}
	// read one
	if _, err = r.r.Read(point[:]); err != nil {
		return
	}
	r.seq++
	return nil
}

// Max returns the expected number of matchable
// this decoder will receive
func (r *Reader) Max() int64 {
	return r.max
}

// init the permutations slice matrix
func initP(n int64) []int64 {
	var p = make([]int64, n)
	var max = big.NewInt(n - 1)
	// Chooses a uniform random int64
	choose := func() int64 {
		i, err := rand.Int(rand.Reader, max)
		if err != nil {
			return 0
		}
		return i.Int64()
	}
	// Initialize a trivial permutation
	for i := int64(0); i < n; i++ {
		p[i] = i
	}
	// and then shuffle it by random swaps
	for i := int64(0); i < n; i++ {
		if j := choose(); j != i {
			p[j], p[i] = p[i], p[j]
		}
	}

	return p
}
