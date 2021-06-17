package ot

import (
	"net"
	"testing"
)

func TestSend(t *testing.T) {
	msgs := make([][2][]byte, 3)
	msgs[0] = [2][]byte{[]byte("m0"), []byte("m1")}
	msgs[1] = [2][]byte{[]byte("secret1"), []byte("secret2")}
	msgs[2] = [2][]byte{[]byte("code1"), []byte("code2")}
	s, _ := NewBaseOt(0, len(msgs), "P256")

	l, err := net.Listen("tcp", "127.0.0.1:10008")
	if err != nil {
		t.Errorf("net listen encountered error: %s", err)
	}
	defer l.Close()

	go func() {
		_, err := net.Dial("tcp", "127.0.0.1:10008")
		if err != nil {
			t.Errorf("Cannot dial: %s", err)
		}
	}()

	conn, err := l.Accept()
	if err != nil {
		t.Errorf("Cannot create connection in listen accept: %s", err)
	}
	defer conn.Close()

	err = s.Send(msgs, conn)
	if err != nil {
		t.Errorf("Send encountered error: %s", err)
	}
}
