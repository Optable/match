package dhpsi

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/bwesterb/go-ristretto"
)

const (
	// EncodedLen is the lenght of one encoded ristretto point
	EncodedLen = 32
	// PrefixedLen is the lenght of one prefixed email identifier
	EmailPrefixedLen = 66
)

var (
	ErrUnexpectedEncodeByte = fmt.Errorf("received a byte to encode past the configured size")
)

type Key struct {
	*ristretto.Scalar
}

type PermutationEncoder interface {
	Encode([]byte) (err error)
	Permutations() []int64
}

type DeriveMultiplyEncoder struct {
	w              io.Writer
	max, seq, sent int64
	r              Ristretto
	// precomputed order to send things in
	permutations []int64
	// buffered in the order received by Encode()
	b [][EncodedLen]byte
}

type MultiplyEncoder struct {
	w        io.Writer
	max, seq int64
	r        Ristretto
}

type Reader struct {
	r        io.Reader
	seq, max int64
}

type MultiplyReader struct {
	r Reader
}

// NewDeriveMultiplyEncoder returns a dhpsi encoder that hashes, encrypts
// and shuffles matchable values on n sequences of bytes to be sent out.
// It first computes a permutation table and subsequently sends out sequences ordered
// by the precomputed permutation table.
//
// This is the first stage of doing a DH exchange.
func NewDeriveMultiplyEncoder(w io.Writer, n int64, r Ristretto) (*DeriveMultiplyEncoder, error) {
	if err := binary.Write(w, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	// and create the encoder
	return &DeriveMultiplyEncoder{w: w, max: n, r: r, permutations: initP(n), b: make([][EncodedLen]byte, n)}, nil

}

// Encode one prefixed ID. First derive and then multiply by the
// precomputed scaler, written out to the underlying writer while following
// the order of permutations created at NewShufflerEncoder.
// Returns ErrUnexpectedEncodeByte when the whole expected sequence has been sent.
func (enc *DeriveMultiplyEncoder) Encode(prefixedID []byte) (err error) {
	// ignore any encode past the max encodes
	// we're configured for
	if enc.seq == enc.max {
		return ErrUnexpectedEncodeByte
	}

	// derive/multiply
	p := enc.r.DeriveMultiply(prefixedID)

	// we follow the permutation matrix and send
	// or cache incoming matchables
	next := enc.permutations[enc.sent]
	if next == enc.seq {
		//  we fall perfectly in sequence, write it out
		_, err = enc.w.Write(p[:])
		enc.sent++
	} else {
		// cache the current sequence
		enc.b[enc.seq] = p
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
func (enc *DeriveMultiplyEncoder) Permutations() []int64 {
	return enc.permutations
}

// InvertedPermutations returns the reverse of the permutation matrix
// that was computed on initialization
func (enc *DeriveMultiplyEncoder) InvertedPermutations() []int64 {
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

// NewEncoder creates an encoder that does the second stage of the DH exchange,
// this time doing a simple scalar multiplication.
func NewMultiplyEncoder(w io.Writer, n int64, r Ristretto) (*MultiplyEncoder, error) {
	// send the max value first
	if err := binary.Write(w, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	return &MultiplyEncoder{w: w, max: n, r: r}, nil
}

// Encode the fixed lenght point by doing a simple multiply
// and write it out to the underlying writer.
// Returns ErrUnexpectedEncodeByte when the whole expected sequence has been sent.
func (enc *MultiplyEncoder) Encode(point [EncodedLen]byte) (err error) {
	// ignore any encode past the max encodes
	// we're configured for
	if enc.seq == enc.max {
		return ErrUnexpectedEncodeByte
	}

	// multiply by our scalar
	b := enc.r.Multiply(point)

	if _, err = enc.w.Write(b[:]); err != nil {
		return err
	}
	enc.seq++
	//
	return
}

// NewReader makes a simple reader that sits on the other end
// of the ShufflerEncoder or the Encoder and reads encoded ristretto hashes.
func NewReader(r io.Reader) (*Reader, error) {
	var max int64
	// extract the max value
	if err := binary.Read(r, binary.LittleEndian, &max); err != nil {
		return nil, err
	}
	return &Reader{r: r, max: max}, nil
}

// Decode a matchable point into p. Returns io.EOF when
// the sequence has been completely read.
func (dec *Reader) Read(p *[EncodedLen]byte) (err error) {
	// ignore any read past the max size
	// we're configured for
	if dec.seq == dec.max {
		return io.EOF
	}
	// read one
	var b []byte = make([]byte, EncodedLen)
	if _, err = dec.r.Read(b); err != nil {
		return
	}
	// one done
	copy(p[:], b)
	dec.seq++
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
