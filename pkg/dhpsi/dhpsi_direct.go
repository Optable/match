package dhpsi

import (
	"encoding/binary"
	"io"
)

type ShufflerDirectEncoder struct {
	w        io.Writer
	seq, max int64
	r        Ristretto
	// precomputed order to send things in
	permutations []int64
	// buffered in the order received by Encode()
	b [][EncodedLen]byte
}

// NewShufflerDirectEncoder returns a dhpsi encoder that hashes, encrypts
// and shuffles matchable values on n sequences of bytes to be sent out.
// It first computes a permutation table and subsequently sends out sequences ordered
// by the precomputed permutation table. This is the first stage of doing a DH exchange.
func NewShufflerDirectEncoder(w io.Writer, n int64, r Ristretto) (*ShufflerDirectEncoder, error) {
	// send the max value first
	if err := binary.Write(w, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	// and create the encoder
	return &ShufflerDirectEncoder{w: w, max: n, r: r, permutations: initP(n), b: make([][EncodedLen]byte, n)}, nil
}

// Encode one prefixed matchable. Hashed, encrypted
// and written out to the underlying writer, following
// the order of permutations created at NewShufflerEncoder.
// Returns io.EOF when the whole expected sequence has been sent.
func (enc *ShufflerDirectEncoder) Encode(matchable []byte) (err error) {
	// ignore any encode past the max encodes
	// we're configured for
	if enc.seq == enc.max {
		return ErrUnexpectedEncodeByte
	}
	// derive/multiply
	p := enc.r.DeriveMultiply(matchable)
	// buffer
	enc.b[enc.seq] = p
	enc.seq++
	// after we processed everything flush the buffer
	if enc.seq == enc.max {
		for _, p := range enc.permutations {
			if _, err = enc.w.Write(enc.b[p][:]); err != nil {
				return
			}
		}
	}
	return
}

// Permutations returns the permutation matrix
// that was computed on initialization
func (enc *ShufflerDirectEncoder) Permutations() []int64 {
	return enc.permutations
}
