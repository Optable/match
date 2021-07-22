package bpsi

import (
	root_bf "github.com/devopsfaith/bloomfilter"
	base_bf "github.com/devopsfaith/bloomfilter/bloomfilter"
)

// FalsePositive is the fixed false positive rate parameter for the bloomfilter,
// expressed in terms of 0-1 is 0% - 100%
const FalsePositive = 1e-6

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

// NewBloomfilter returns a new bloomfilter able to
// contain n items with a ProbCollide chance of collision (0-1: 0% to 100%)
func NewBloomfilter(n int64) bloomfilter {
	var bf = base_bf.New(root_bf.Config{N: (uint)(max(n, 1)), P: FalsePositive, HashName: root_bf.HASHER_OPTIMAL})
	return devopsfaith{bf: bf}
}

// Add an identifier to a devopsfaith bloomfilter
func (bf devopsfaith) Add(identifier []byte) {
	bf.bf.Add(identifier)
}

// Check for the presence of an identifier in the bloomfilter
func (bf devopsfaith) Check(identifier []byte) bool {
	return bf.bf.Check(identifier)
}

// MarshalBinary the entire bloomfilter and return the bytes
func (bf devopsfaith) MarshalBinary() ([]byte, error) {
	return bf.bf.MarshalBinary()
}

// UnmarshalBinary b into a new bloomfilter
func UnmarshalBinary(b []byte) (bloomfilter, error) {
	var bf = &base_bf.Bloomfilter{}
	if err := bf.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return devopsfaith{bf: bf}, nil
}

// max would be great in the stdlib
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
