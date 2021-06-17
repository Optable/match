package ot

import (
	"net"
	"testing"
)

var (
	network = "tcp"
	address = "127.0.0.1:10008"
	msgs    = [][2][]byte{
		{[]byte("m0"), []byte("m1")},
		{[]byte("secret1"), []byte("secret2")},
		{[]byte("code1"), []byte("code2")},
	}
	choices = []uint8{0, 1, 1}
	curve   = "P256"
)

func TestSimplest(t *testing.T) {
	ss, err := NewBaseOt(1, len(msgs), curve)
	if err != nil {
		t.Errorf("Error creating simplest OT: %s", err)
	}

	l, err := net.Listen(network, address)
	if err != nil {
		t.Errorf("net listen encountered error: %s", err)
	}
	defer l.Close()

	go func() {
		scon, err := net.Dial(network, address)
		if err != nil {
			t.Errorf("Cannot dial: %s", err)
		}
		err = ss.Send(msgs, scon)
		if err != nil {
			t.Errorf("Send encountered error: %s", err)
		}
	}()

	conn, err := l.Accept()
	if err != nil {
		t.Errorf("Cannot create connection in listen accept: %s", err)
	}
	defer conn.Close()

	sr, err := NewBaseOt(1, len(choices), curve)
	msg := make([][]byte, len(choices))
	err = sr.Receive(choices, msg, conn)
	if err != nil {
		t.Errorf("Receive encountered error: %s", err)
	}
}
