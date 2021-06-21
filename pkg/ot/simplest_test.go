package ot

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"
)

var (
	network   = "tcp"
	address   = "127.0.0.1:"
	curve     = "P256"
	baseCount = 256
)

func genMsg(n int) [][2][]byte {
	rand.Seed(time.Now().UnixNano())
	data := make([][2][]byte, n)
	for i := 0; i < n; i++ {
		for j, _ := range data[i] {
			data[i][j] = make([]byte, 64)
			rand.Read(data[i][j])
		}
	}

	return data
}

func genChoiceBits(n int) []uint8 {
	choices := make([]uint8, n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i, _ := range choices {
		choices[i] = uint8(r.Intn(2))
	}

	return choices
}

func initReceiver(msgLen []int, choices []uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}
		go receiveHandler(conn, msgLen, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func receiveHandler(conn net.Conn, msgLen []int, choices []uint8, msgBus chan<- []byte, errs chan<- error) {
	defer close(msgBus)

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
	msgs := genMsg(baseCount)
	msgLen := make([]int, len(msgs))
	for i, m := range msgs {
		msgLen[i] = len(m[0])
	}

	choices := genChoiceBits(baseCount)

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	addr, err := initReceiver(msgLen, choices, msgBus, errs)
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
		t.Fatal("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if string(m) != string(msgs[i][choices[i]]) {
			t.Fatalf("OT failed, want to receive msg: %s, got: %s", msgs[i][choices[i]], m)
		}
	}
}
