package npsi

import (
	"encoding/binary"
	"io"

	"github.com/optable/match/internal/hash"
)

type hashPair struct {
	x []byte
	h uint64
}

// Read one hash
func HashRead(r io.Reader, u *uint64) error {
	return binary.Read(r, binary.BigEndian, &u)
}

// Write one hash
func HashWrite(w io.Writer, u uint64) error {
	return binary.Write(w, binary.BigEndian, u)
}

// ReadAll from r until io.EOF and write into a
// channel.
// note that binary.Read will return EOF only if no bytes
// are read and if an EOF happens after reading some but not all the bytes,
// Read returns ErrUnexpectedEOF.
func ReadAll(r io.Reader) <-chan uint64 {
	var out = make(chan uint64)
	go func() {
		defer close(out)
		for {
			var u uint64
			if err := HashRead(r, &u); err == nil {
				out <- u
			} else {
				return
			}
		}
	}()
	return out
}

// HashAll reads all identifiers from identifiers
// and hashes them until identifiers closes
func HashAll(h hash.Hasher, identifiers <-chan []byte) <-chan hashPair {
	var pairs = make(chan hashPair)

	// just read and hash baby
	go func() {
		for identifier := range identifiers {
			h := h.Hash64(identifier)
			pairs <- hashPair{x: identifier, h: h}
		}
	}()
	return pairs
}
