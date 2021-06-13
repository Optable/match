package permutations

type null int

func NewNil(n int64) (null, error) {
	return 0, nil
}

// Shuffle using the naive method
// with n the number to permute/the index of the permutation vector.
func (k null) Shuffle(n int64) int64 {
	return n
}
