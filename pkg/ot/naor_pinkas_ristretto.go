package ot

import (
	"io"
)

type naorPinkasRistretto struct {
	baseCount int
	msgLen    []int
}

func newNaorPinkasRistretto(baseCount int, msgLen []int) (naorPinkasRistretto, error) {
	if len(msgLen) != baseCount {
		return naorPinkasRistretto{}, ErrBaseCountMissMatch
	}
	return naorPinkasRistretto{baseCount: baseCount, msgLen: msgLen}, nil
}

func (n naorPinkasRistretto) Send(messages [][2][]byte, rw io.ReadWriter) error {
	return nil
}

func (n naorPinkasRistretto) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error {
	return nil
}
