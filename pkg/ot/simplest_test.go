package ot

import (
	"testing"
)

func TestSend(t *testing.T) {
	msgs := make([][2]string, 3)
	msgs[0] = [2]string{"m0", "m1"}
	msgs[1] = [2]string{"secret1", "secret2"}
	msgs[2] = [2]string{"code1", "code2"}
	s, _ := NewBaseOt(0, len(msgs))
	c := make(chan []byte)
	s.Send(msgs, c)
}
