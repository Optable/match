package ot

import (
	"crypto/elliptic"
	"io"
)

type naorPinkas struct {
	baseCount int
	curve     elliptic.Curve
	encodeLen int
	msgLen    []int
}

func newNaorPinkas(baseCount int, curveName string, msgLen []int) (naorPinkas, error) {
	if len(msgLen) != baseCount {
		return naorPinkas{}, ErrBaseCountMissMatch
	}
	curve, encodeLen := initCurve(curveName)
	return naorPinkas{baseCount: baseCount, curve: curve, encodeLen: encodeLen, msgLen: msgLen}, nil
}

func (n naorPinkas) Send(messages [][2][]byte, rw io.ReadWriter) error {
	return nil
}

func (n naorPinkas) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error {
	return nil
}
