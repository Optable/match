package cuckoo

import (
	"math"
	"testing"
)

func TestFindStashSize(t *testing.T) {
	size := uint64(math.Pow(2, 8)) - 1
	sSize := findStashSize(size)
	if sSize != 12 {
		t.Errorf("findStashSize(255) = %d; want 12", sSize)
	}

	size = uint64(math.Pow(2, 12)) - 1
	sSize = findStashSize(size)
	if sSize != 6 {
		t.Errorf("findStashSize(255) = %d; want 6", sSize)
	}

	size = uint64(math.Pow(2, 16)) - 1
	sSize = findStashSize(size)
	if sSize != 4 {
		t.Errorf("findStashSize(255) = %d; want 4", sSize)
	}

	size = uint64(math.Pow(2, 20)) - 1
	sSize = findStashSize(size)
	if sSize != 3 {
		t.Errorf("findStashSize(255) = %d; want 12", sSize)
	}

	size = uint64(math.Pow(2, 24)) - 1
	sSize = findStashSize(size)
	if sSize != 2 {
		t.Errorf("findStashSize(255) = %d; want 12", sSize)
	}

	size = uint64(math.Pow(2, 24)) + 1
	sSize = findStashSize(size)
	if sSize != 0 {
		t.Errorf("findStashSize(255) = %d; want 0", sSize)
	}
}
