package util

import (
	"bufio"
	"io"
)

// Count counts the number of lines in a file
func Count(r io.Reader) (int64, error) {
	var n int64
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		n++
	}
	if err := scanner.Err(); err != nil {
		return n, err
	}

	return n, nil
}
