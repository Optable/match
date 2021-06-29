package ot

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))
var s = make([]uint8, 1000)

func BenchmarkSampleBitSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := sampleBitSlice(r, s)
		if err != nil {
			b.Log(err)
		}
	}
}

func initIknpReceiver(ristretto bool, msgLen []int, choices []uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}

		go iknpReceiveHandler(conn, ristretto, msgLen, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func iknpReceiveHandler(conn net.Conn, ristretto bool, msgLen []int, choices []uint8, msgBus chan<- []byte, errs chan<- error) {
	defer close(msgBus)

	sr, err := NewIknp(3, 128, Simplest, ristretto, msgLen)
	if err != nil {
		errs <- err
	}

	msg := make([][]byte, 3)
	err = sr.Receive(choices, msg, conn)
	if err != nil {
		errs <- err
	}

	for _, m := range msg {
		msgBus <- m
	}
}

func TestIknpOtExtension(t *testing.T) {
	msgsIknp := make([][2][]byte, 3)
	msgsIknp[0] = [2][]byte{
		[]byte("Test 1"),
		[]byte("Test 2"),
	}

	msgsIknp[1] = [2][]byte{
		[]byte("Simplest OT"),
		[]byte("Naor-pinkas"),
	}

	msgsIknp[2] = [2][]byte{
		[]byte("IKNP OT extension"),
		[]byte("blavlablalvlalsda"),
	}

	msgLenIknp := make([]int, len(msgsIknp))
	for i, m := range msgsIknp {
		msgLenIknp[i] = len(m[0])
	}

	choicesIknp := []uint8{1, 1, 1}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()

	addr, err := initIknpReceiver(false, msgLenIknp, choicesIknp, msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		ss, err := NewIknp(3, 128, Simplest, false, msgLen)
		if err != nil {
			errs <- fmt.Errorf("Error creating IKNP OT: %s", err)
		}

		err = ss.Send(msgsIknp, conn)
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
	t.Logf("Time taken for IKNP OT of %d OTs is: %v\n", 3, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("OT failed, did not receive any messages")
	}

	for i := 0; i < 3; i++ {
		t.Log(string(msg[i]))
	}

	for i, m := range msg {
		if bytes.Compare(m, msgsIknp[i][choicesIknp[i]]) != 0 {
			t.Fatalf("OT failed got: %v, want %v", m, msgsIknp[i][choicesIknp[i]])
		}
	}
}
