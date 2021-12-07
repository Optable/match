package permutations

// Permutations is an interface satisfied by anything with a proper
// Shuffle method
type Permutations interface {
	Shuffle(n int64) int64
}

const (
	Kensler = iota
	Naive
	Nil
)

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
