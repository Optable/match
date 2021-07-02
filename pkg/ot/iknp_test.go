package ot

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

func BenchmarkSampleBitSlice(b *testing.B) {
	s := make([]uint8, 1)
	for i := 0; i < b.N; i++ {
		sampleBitSlice(r, s)
	}
}

func initIknpReceiver(ot Ot, choices []uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}

		go iknpReceiveHandler(conn, ot, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func iknpReceiveHandler(conn net.Conn, ot Ot, choices []uint8, msgBus chan<- []byte, errs chan<- error) {
	defer close(msgBus)

	msg := make([][]byte, baseCount)
	err := ot.Receive(choices, msg, conn)
	if err != nil {
		errs <- err
	}

	for _, m := range msg {
		msgBus <- m
	}
}

func TestIknpOtExtension(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	ot, err := NewIknp(baseCount, 256, Simplest, false, msgLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initIknpReceiver(ot, choices, msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		if err != nil {
			errs <- fmt.Errorf("Error creating IKNP OT: %s", err)
		}

		err = ot.Send(messages, conn)
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

	// stop timer
	end := time.Now()
	t.Logf("Time taken for IKNP OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("IKNP OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if bytes.Compare(m, messages[i][choices[i]]) != 0 {
			t.Fatalf("IKNP OT failed got: %v, want %v", m, messages[i][choices[i]])
		}
	}
}

func TestImprovedIknpOtExtension(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	ot, err := NewImprovedIknp(baseCount, 256, Simplest, false, msgLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initIknpReceiver(ot, choices, msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		if err != nil {
			errs <- fmt.Errorf("Error creating IKNP OT: %s", err)
		}

		err = ot.Send(messages, conn)
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

	// stop timer
	end := time.Now()
	t.Logf("Time taken for Improved IKNP OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("Improved IKNP OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if bytes.Compare(m, messages[i][choices[i]]) != 0 {
			t.Fatalf("Improved IKNP OT failed at meesage %d, got: %v, want %v from %v", i, m, messages[i][choices[i]], messages[i])
		}
	}
}
