package ot

import (
	"crypto/elliptic"
)

type naorPinkas struct {
	baseCount int
	curve     elliptic.Curve
}

func NewNaorPinkas(baseCount int, curveName string) (*naorPinkas, error) {
	return &naorPinkas{baseCount: baseCount, curve: InitCurve(curveName)}, nil
}

func (n *naorPinkas) Send(messages [][2][]byte, c chan []byte) error {
	return nil
}

func (n *naorPinkas) Receive(choices []uint8, messages [][]byte, c chan []byte) error {
	return nil
}
