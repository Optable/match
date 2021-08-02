package bpsi

import (
	"encoding/binary"
	"fmt"
	"io"

	bloom "github.com/bits-and-blooms/bloom/v3"
	root_bf "github.com/devopsfaith/bloomfilter"
	base_bf "github.com/devopsfaith/bloomfilter/bloomfilter"
)

const (
	// FalsePositive is the fixed false positive rate parameter for the bloomfilter,
	// expressed in terms of 0-1 is 0% - 100%
	FalsePositive = 1e-6

	BloomfilterTypeDevopsfaith = iota
	BloomfilterTypeBitsAndBloom
)

type Bloomfilter int

var (
	BloomfilterDevopsfaith  Bloomfilter = BloomfilterTypeDevopsfaith
	BloomfilterBitsAndBloom Bloomfilter = BloomfilterTypeBitsAndBloom
)

// bloomfilter type to wrap around
// an actual implementation
type bloomfilter interface {
	Add(identifier []byte)
	Check(identifier []byte) bool
	WriteTo(rw io.Writer) (int64, error)
}

// devopsfaith implementation of the
// bloomfilter interface
type devopsfaith struct {
	bf *base_bf.Bloomfilter
}

// bits-and-bloom implementation of the
// bloomfilter interface
type bitsAndBloom struct {
	bf *bloom.BloomFilter
}

// NewBloomfilter instantiates a bloomfilter
// with the given type and number of items to be inserted.
func NewBloomfilter(t Bloomfilter, n int64) (bloomfilter, error) {
	switch t {
	case BloomfilterTypeDevopsfaith:
		var bf = base_bf.New(root_bf.Config{N: (uint)(max(n, 1)), P: FalsePositive, HashName: root_bf.HASHER_OPTIMAL})
		return devopsfaith{bf: bf}, nil
	case BloomfilterTypeBitsAndBloom:
		return bitsAndBloom{bf: bloom.NewWithEstimates(uint(n), FalsePositive)}, nil
	default:
		return nil, fmt.Errorf("unsupported bloomfilter type %d", t)
	}
}

// Add an identifier to a devopsfaith bloomfilter
func (bf devopsfaith) Add(identifier []byte) {
	bf.bf.Add(identifier)
}

// Check for the presence of an identifier in the bloomfilter
func (bf devopsfaith) Check(identifier []byte) bool {
	return bf.bf.Check(identifier)
}

// WriteTo marshals the entire bloomfilter to rw
func (bf devopsfaith) WriteTo(rw io.Writer) (int64, error) {
	if b, err := bf.bf.MarshalBinary(); err == nil {
		l := int64(len(b))
		if err := binary.Write(rw, binary.BigEndian, l); err != nil {
			return 0, err
		}
		n, err := rw.Write(b)
		return int64(n), err
	} else {
		return 0, err
	}
}

// Add an identifier to a bitsAndBloom bloomfilter
func (bf bitsAndBloom) Add(identifier []byte) {
	bf.bf.Add(identifier)
}

// Check for the presence of an identifier in the bloomfilter
func (bf bitsAndBloom) Check(identifier []byte) bool {
	return bf.bf.Test(identifier)
}

// MarshalBinary marshals the entire bloomfilter and return the bytes
func (bf bitsAndBloom) WriteTo(rw io.Writer) (int64, error) {
	return bf.bf.WriteTo(rw)
}

// ReadFrom r into a new bitsAndBloom bloomfilter
func ReadFrom(r io.Reader) (bloomfilter, int64, error) {
	var bf = &bloom.BloomFilter{}
	n, err := bf.ReadFrom(r)
	return bitsAndBloom{bf: bf}, n, err
}

// max would be great in the stdlib
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
