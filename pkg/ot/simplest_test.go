package ot

import (
	"testing"
)

func TestSend(t *testing.T) {
	msgs := make([][2][]byte, 3)
	msgs[0] = [2][]byte{[]byte("m0"), []byte("m1")}
	msgs[1] = [2][]byte{[]byte("secret1"), []byte("secret2")}
	msgs[2] = [2][]byte{[]byte("code1"), []byte("code2")}
	s, _ := NewBaseOt(0, len(msgs), "P256")
	c := make(chan []byte)
	s.Send(msgs, c)
}
