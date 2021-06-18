package ot

import (
	"fmt"
	"net"
	"testing"
)

var (
	network = "tcp"
	address = "127.0.0.1:"
	msgs    = [][2][]byte{
		{[]byte("m0"), []byte("m1")},
		{[]byte("secret1"), []byte("secret2")},
		{[]byte("code1"), []byte("code2")},
	}
	choices = []uint8{0, 1, 1}
	curve   = "P256"
)

func initReceiver(msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}
		go receiveHandler(conn, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func receiveHandler(conn net.Conn, msgBus chan<- []byte, errs chan<- error) {
	defer close(msgBus)
	msgLen := make([]int, len(msgs))
	for i, m := range msgs {
		msgLen[i] = len(m[0])
	}

	sr, err := NewBaseOt(1, len(choices), curve, msgLen)
	if err != nil {
		errs <- err
	}

	msg := make([][]byte, len(choices))
	err = sr.Receive(choices, msg, conn)
	if err != nil {
		errs <- err
	}

	for _, m := range msg {
		msgBus <- m
	}
}

func TestSimplestOt(t *testing.T) {
	msgLen := make([]int, len(msgs))
	for i, m := range msgs {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	addr, err := initReceiver(msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		ss, err := NewBaseOt(1, len(msgs), curve, msgLen)
		if err != nil {
			errs <- fmt.Errorf("Error creating simplest OT: %s", err)
		}

		err = ss.Send(msgs, conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(msgBus)
		}

	}()

	// Receive msg
	var msg [][]byte
	for m := range msgBus {
		msg = append(msg, m)
	}

	//errors?
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatalf("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if string(m) != string(msgs[i][choices[i]]) {
			t.Fatalf("OT failed, want to receive msg: %s, got: %s", string(msgs[i][choices[i]]), string(m))
		}
	}
}
