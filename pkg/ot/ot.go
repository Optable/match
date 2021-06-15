package ot

import (
	"fmt"
	"golang.org/x/crypto/curve25519"
)

const (
	NaorPinkas = iota
	Simplest
)

var (
	ErrUnknownOt          = fmt.Errorf("cannot create an Ot that follows an unknown protocol")
	ErrBaseCountMissMatch = fmt.Errorf("provided slices is not the same length as the number of base OT.")
)

// OT implements different BaseOT
type Ot interface {
	Send(messages [][2]string, c chan []byte) error
	Receive(choices []uint8, messages []string, c chan []byte) error
}

// NewBaseOt returns an Ot of type t
func NewBaseOt(t int, baseCount int) (Ot, error) {
	switch t {
	case NaorPinkas:
		return NewNaorPinkas(baseCount)
	case Simplest:
		return NewSimplest(baseCount)
	default:
		return nil, ErrUnknownOt
	}
}

func genSecretKey() (secret []byte, err error) {
	secret := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	return
}

func genPublicKey(secret []byte) (public []byte, err error) {
	if public, err := curve25519.x25519(secret, curve25519.Basepoint); err != nil {
		return nil, err
	}
	return
}
