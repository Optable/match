package permutations

type null int

// Shuffle using the nil method
// just return the same value
func (k null) Shuffle(n int64) int64 {
	return n
}
