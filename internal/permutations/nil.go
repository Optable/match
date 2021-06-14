package permutations

type null int

func NewNil(n int64) (null, error) {
	return 0, nil
}

// Shuffle using the nil method
// just return the same value
func (k null) Shuffle(n int64) int64 {
	return n
}
