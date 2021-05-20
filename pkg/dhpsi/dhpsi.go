package dhpsi

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
)

const (
	// EncodedLen is the lenght of one encoded ristretto point
	EncodedLen = 32
	// PrefixedLen is the lenght of one prefixed email identifier
	EmailPrefixedLen = 66
)

var (
	ErrUnexpectedEncodeByte = fmt.Errorf("received a byte to encode past the configured encoder size")
)

//
// Writers
//

// types
type DeriveMultiplyShuffler struct {
	w              io.Writer
	max, seq, sent int64
	gr             Ristretto
	// precomputed order to send things in
	permutations []int64
	// buffered in the order received by Shuffle()
	b [][EncodedLen]byte
}

type Writer struct {
	w        io.Writer
	max, seq int64
}

// NewDeriveMultiplyShuffler returns a dhpsi encoder that hashes, encrypts
// and shuffles matchable values on n sequences of bytes to be sent out.
// It first computes a permutation table and subsequently sends out sequences ordered
// by the precomputed permutation table.
//
// This is the first stage of doing a DHPSI exchange.
func NewDeriveMultiplyShuffler(w io.Writer, n int64, gr Ristretto) (*DeriveMultiplyShuffler, error) {
	if err := binary.Write(w, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	// and create the encoder
	return &DeriveMultiplyShuffler{w: w, max: n, gr: gr, permutations: initP(n), b: make([][EncodedLen]byte, n)}, nil

}

// Shuffle one identifier. First derive and then multiply by the
// precomputed scaler, written out to the underlying writer while following
// the order of permutations created at NewDeriveMultiplyShuffler.
// Returns ErrUnexpectedEncodeByte when the whole expected sequence has been sent.
func (enc *DeriveMultiplyShuffler) Shuffle(identifier []byte) (err error) {
	// ignore any encode past the max encodes
	// we're configured for
	if enc.seq == enc.max {
		return ErrUnexpectedEncodeByte
	}

	// derive/multiply
	point := enc.gr.DeriveMultiply(identifier)

	// we follow the permutation matrix and send
	// or cache incoming matchables
	next := enc.permutations[enc.sent]
	if next == enc.seq {
		//  we fall perfectly in sequence, write it out
		_, err = enc.w.Write(point[:])
		enc.sent++
	} else {
		// cache the current sequence
		enc.b[enc.seq] = point
	}
	enc.seq++
	// after we processed everything we will very probably
	// have cached hashes left to send.
	// flush the buffer, in enc.permutations order
	if enc.seq == enc.max {
		for _, pos := range enc.permutations[enc.sent:] {
			if _, err = enc.w.Write(enc.b[pos][:]); err != nil {
				return
			}
		}
	}
	return
}

// Permutations returns the permutation matrix
// that was computed on initialization
func (enc *DeriveMultiplyShuffler) Permutations() []int64 {
	return enc.permutations
}

// InvertedPermutations returns the reverse of the permutation matrix
// that was computed on initialization
func (enc *DeriveMultiplyShuffler) InvertedPermutations() []int64 {
	return invertedPermutations(enc.permutations)
}

// InvertedPermutations returns the reverse of the permutation matrix
// that was computed on initialization
func invertedPermutations(in []int64) []int64 {
	var invertedpermutations = make([]int64, len(in))
	for i := 0; i < len(invertedpermutations); i++ {
		invertedpermutations[in[i]] = int64(i)
	}
	return invertedpermutations
}

// NewWriter creates a writer that first sends out
// the total number of items that will be sent out
func NewWriter(w io.Writer, n int64) (*Writer, error) {
	// send the max value first
	if err := binary.Write(w, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	return &Writer{w: w, max: n}, nil
}

// Write out the fixed length point to the underlying writer
// while sequencing
func (w *Writer) Write(point [EncodedLen]byte) (err error) {
	// ignore any encode past the max encodes
	// we're configured for
	if w.seq == w.max {
		return ErrUnexpectedEncodeByte
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

// types
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
// multiplies them using gr
func NewMultiplyReader(r io.Reader, gr Ristretto) (*MultiplyReader, error) {
	if r, err := NewReader(r); err != nil {
		return nil, err
	} else {
		return &MultiplyReader{r: r, gr: gr}, nil
	}
}

// Multiply a point from the underyling reader with ristretto
// and write it into p. Returns io.EOF when
// the sequence has been completely read.
func (r *MultiplyReader) Multiply(point *[EncodedLen]byte) (err error) {
	var b [EncodedLen]byte
	if err := r.r.Read(&b); err != nil {
		return err
	} else {
		*point = r.gr.Multiply(b)
		return nil
	}

}

// Max is the expected number of matchable
// this decoder will receive
func (dec *MultiplyReader) Max() int64 {
	return dec.r.max
}

// NewReader makes a simple reader that sits on the other end
// of the DeriveMultiplyShuffler or the Writer and reads encoded ristretto hashes
func NewReader(r io.Reader) (*Reader, error) {
	var max int64
	// extract the max value
	if err := binary.Read(r, binary.LittleEndian, &max); err != nil {
		return nil, err
	}
	return &Reader{r: r, max: max}, nil
}

// Read a point from the underyling reader and
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

// Max is the expected number of matchable
// this decoder will receive
func (dec *Reader) Max() int64 {
	return dec.max
}

// init the permutations slice matrix
func initP(n int64) []int64 {
	var p = make([]int64, n)
	var max = big.NewInt(n - 1)
	// Chooses a uniform random int64
	choose := func() int64 {
		if i, err := rand.Int(rand.Reader, max); err == nil {
			return i.Int64()
		} else {
			return 0
		}
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
