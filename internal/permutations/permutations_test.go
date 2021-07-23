package permutations

import "testing"

const xxx = 10

func TestModelSingle(t *testing.T) {
	var seq, sent int64
	var max int64 = xxx
	// make a cache
	var cache = make([]int64, xxx)
	// create the permutations
	p, _ := NewKensler(xxx)
	// create the sequence to compare to
	shuffle := genSequence(p)
	// output
	var output = make([]int64, xxx)
	// input
	var input = make([]int64, xxx)

	// run it the same way the single threaded shuffler runs it
	for i := 0; i < xxx; i++ {
		input[i] = int64(i)
		next := p.Shuffle(sent)

		if next == seq {
			//  we fall perfectly in sequence, write it out
			output[sent] = int64(i)
			sent++
		} else {
			// cache the current sequence
			cache[seq] = int64(i)
		}
		seq++

		if seq == max {
			for i := sent; i < max; i++ {
				pos := p.Shuffle(i)
				output[i] = cache[pos]
			}
		}
	}

	// output & sequence should be the same
	for k, v := range output {
		if v != shuffle[k] {
			t.Errorf("expected %d got %d", v, shuffle[k])
		}
	}

	// should be able to reverse the sequence
	for k, v := range output {
		pos := p.Shuffle(int64(k))
		if v != input[pos] {
			t.Errorf("expected %d got %d", v, input[pos])
		}
	}
}

func genSequence(p Permutations) (data []int64) {
	for i := int64(0); i < xxx; i++ {
		data = append(data, p.Shuffle(i))
	}
	return
}
