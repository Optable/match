package kkrtpsi

import (
	"encoding/binary"
	"io"

	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
)

const K = 512

// findK returns the number of base OT for OPRF
// these numbers are from the paper: Efficient Batched Oblivious PRF with Applications to Private Set Intersection
// by Vladimir Kolesnikov, Ranjit Kumaresan, Mike Rosulek, and Ni Treu in 2016.
// Reference:	http://dx.doi.org/10.1145/2976749.2978381 (KKRT)
func findK(n int64) int {
	switch {
	// 2^8
	case n > 0 && n <= 256:
		return 424
	// 2^12
	case n > 256 && n <= 4096:
		return 432
	// 2^16
	case n > 4096 && n <= 65536:
		return 440
	// 2^20
	case n > 65536:
		return 448
	default:
		return 128
	}
}

// HashAll reads all inputs from inputs
// and hashes them until inputs closes
func HashAll(h hash.Hasher, input <-chan []byte) <-chan uint64 {
	var hashes = make(chan uint64)

	// just read and hash
	go func() {
		defer close(hashes)
		for in := range input {
			hashes <- h.Hash64(in)
		}
	}()
	return hashes
}

// HashRead reads one hash
func EncodesRead(r io.Reader, u *[cuckoo.Nhash]uint64) (err error) {
	err = binary.Read(r, binary.BigEndian, u)
	return
}

// HashWrite writes one hash out
func EncodesWrite(w io.Writer, u [cuckoo.Nhash]uint64) error {
	return binary.Write(w, binary.BigEndian, u)
}
