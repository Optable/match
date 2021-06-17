package ot

import (
	"crypto/elliptic"
	"io"
)

type naorPinkas struct {
	baseCount int
	curve     elliptic.Curve
	encodeLen int
}

func NewNaorPinkas(baseCount int, curveName string) (naorPinkas, error) {
	curve := InitCurve(curveName)
	encodeLen := len(elliptic.Marshal(curve, curve.Params().Gx, curve.Params().Gy))
	return naorPinkas{baseCount: baseCount, curve: curve, encodeLen: encodeLen}, nil
}

func (n naorPinkas) Send(messages [][2][]byte, rw io.ReadWriter) error {
	return nil
}

func (n naorPinkas) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) error {
	return nil
}
