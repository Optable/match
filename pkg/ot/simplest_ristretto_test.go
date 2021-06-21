package ot

import (
	"fmt"
	"net"
	"testing"
)

func initReceiverRistretto(msgLen []int, choices []uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}
		go receiveHandlerRistretto(conn, msgLen, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func receiveHandlerRistretto(conn net.Conn, msgLen []int, choices []uint8, msgBus chan<- []byte, errs chan<- error) {
	defer close(msgBus)

	sr, err := NewBaseOtRistretto(1, len(choices), msgLen)
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

func TestSimplestRistretto(t *testing.T) {
	msgs := genMsg(baseCount)
	msgLen := make([]int, len(msgs))
	for i, m := range msgs {
		msgLen[i] = len(m[0])
	}

	choices := genChoiceBits(baseCount)

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	addr, err := initReceiverRistretto(msgLen, choices, msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		ss, err := NewBaseOtRistretto(1, len(msgs), msgLen)
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
		t.Fatal("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if string(m) != string(msgs[i][choices[i]]) {
			t.Fatalf("OT failed, want to receive msg: %s, got: %s", msgs[i][choices[i]], m)
		}
	}
}
