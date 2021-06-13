package permutations

type Permutations interface {
	Shuffle(n int64) int64
}

const (
	Kensler = iota
	Naive
	Nil
)
