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

// HashRead reads one hash
func HashRead(r io.Reader, u *uint64) (err error) {
	err = binary.Read(r, binary.BigEndian, u)
	return
}

// HashWrite writes one hash out
func HashWrite(w io.Writer, u uint64) error {
	return binary.Write(w, binary.BigEndian, u)
}

// ReadAll reads from r until io.EOF and writes into a channel.
// note that binary.Read will return EOF only if no bytes
// are read and if an EOF happens after reading some but not all the bytes,
// Read returns ErrUnexpectedEOF.
func ReadAll(r io.Reader, n int64) <-chan uint64 {
	var out = make(chan uint64)
	go func() {
		defer close(out)
		for i := int64(0); i < n; i++ {
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
		defer close(pairs)
		for identifier := range identifiers {
			h := h.Hash64(identifier)
			pairs <- hashPair{x: identifier, h: h}
		}
	}()
	return pairs
}
