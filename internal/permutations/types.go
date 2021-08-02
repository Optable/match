package permutations

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
