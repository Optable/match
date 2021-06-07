package cuckoo

import (
	"math"
	"testing"
)

func TestFindStashSize(t *testing.T) {
	size := uint64(math.Pow(2, 8)) - 1
	sSize := findStashSize(size)
	if sSize != 12 {
		t.Errorf("findStashSize(%d) = %d; want 12", size, sSize)
	}

	size = uint64(math.Pow(2, 12)) - 1
	sSize = findStashSize(size)
	if sSize != 6 {
		t.Errorf("findStashSize(%d) = %d; want 6", size, sSize)
	}

	size = uint64(math.Pow(2, 16)) - 1
	sSize = findStashSize(size)
	if sSize != 4 {
		t.Errorf("findStashSize(%d) = %d; want 4", size, sSize)
	}

	size = uint64(math.Pow(2, 20)) - 1
	sSize = findStashSize(size)
	if sSize != 3 {
		t.Errorf("findStashSize(%d) = %d; want 3", size, sSize)
	}

	size = uint64(math.Pow(2, 24)) - 1
	sSize = findStashSize(size)
	if sSize != 2 {
		t.Errorf("findStashSize(%d) = %d; want 2", size, sSize)
	}

	size = uint64(math.Pow(2, 25))
	sSize = findStashSize(size)
	if sSize != 0 {
		t.Errorf("findStashSize(%d) = %d; want 0", size, sSize)
	}
}
