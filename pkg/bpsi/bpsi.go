package bpsi

import (
	"fmt"

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

// bloomfilter type to wrap around
// an actual implementation
type bloomfilter interface {
	Add(identifier []byte)
	Check(identifier []byte) bool
	MarshalBinary() ([]byte, error)
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
func NewBloomfilter(t int, n int64) (bloomfilter, error) {
	switch t {
	case BloomfilterTypeDevopsfaith:
		var bf = base_bf.New(root_bf.Config{N: (uint)(max(n, 1)), P: FalsePositive, HashName: root_bf.HASHER_OPTIMAL})
		return devopsfaith{bf: bf}, nil
	case BloomfilterTypeBitsAndBloom:
		return bitsAndBloom{bf: bloom.NewWithEstimates(uint(n), FalsePositive)}, nil
	default:
		return nil, fmt.Errorf("unsupported ristretto type %d", t)
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

// MarshalBinary marshals the entire bloomfilter and return the bytes
func (bf devopsfaith) MarshalBinary() ([]byte, error) {
	return bf.bf.MarshalBinary()
}

// Add an identifier to a devopsfaith bloomfilter
func (bf bitsAndBloom) Add(identifier []byte) {
	bf.bf.Add(identifier)
}

// Check for the presence of an identifier in the bloomfilter
func (bf bitsAndBloom) Check(identifier []byte) bool {
	return bf.bf.Test(identifier)
}

// MarshalBinary marshals the entire bloomfilter and return the bytes
func (bf bitsAndBloom) MarshalBinary() ([]byte, error) {
	return bf.bf.MarshalJSON()
}

// UnmarshalBinary b into a new bloomfilter
func UnmarshalBinary(b []byte) (bloomfilter, error) {
	var bf = &base_bf.Bloomfilter{}
	if err := bf.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return devopsfaith{bf: bf}, nil
}

// UnmarshalJSON unmarshals b into a new bloomfilter (this needs to be chanegd to merge with UnmarshalBinary above)
func UnmarshalJSON(b []byte) (bloomfilter, error) {
	var bf = &bloom.BloomFilter{}
	if err := bf.UnmarshalJSON(b); err != nil {
		return nil, err
	}
	return bitsAndBloom{bf: bf}, nil
}

// max would be great in the stdlib
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
