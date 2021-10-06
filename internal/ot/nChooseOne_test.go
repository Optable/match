package ot

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/optable/match/internal/crypto"
)

var k = 512

func initKKRTReceiver(ot OT, choices []uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}

		go kkrtReceiveHandler(conn, ot, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func kkrtReceiveHandler(conn net.Conn, ot OT, choices []uint8, msgBus chan<- []byte, errs chan<- error) {
	defer close(msgBus)

	msg := make([][]byte, otExtensionCount)
	err := ot.Receive(choices, msg, conn)
	if err != nil {
		errs <- err
	}

	for _, m := range msg {
		msgBus <- m
	}
}

func TestKKRT(t *testing.T) {
	// sample integer choices
	cc := make([]byte, otExtensionCount)
	for i := range cc {
		cc[i] = byte(r.Intn(tuple))
	}

	mLen := make([]int, otExtensionCount)
	for i, m := range nchooseOneMessages {
		mLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	ot, err := NewKKRT(otExtensionCount, k, tuple, NaorPinkas, false, mLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initKKRTReceiver(ot, cc, msgBus, errs)
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

		ot, err := NewKKRT(otExtensionCount, k, tuple, NaorPinkas, false, mLen)
		if err != nil {
			errs <- err
		}

		err = ot.Send(nchooseOneMessages, conn)
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
	t.Logf("Time taken for KKRT OT of %d OTs is: %v\n", otExtensionCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("KKRT OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if !bytes.Equal(m, nchooseOneMessages[i][cc[i]]) {
			t.Logf("choice[%d]=%d\nmessages=%v\n", i, cc[i], nchooseOneMessages[i])
			t.Fatalf("KKRT OT at msg %d, failed got: %v, want %v", i, m, nchooseOneMessages[i][cc[i]])
		}
	}
}

func TestImprovedKKRT(t *testing.T) {
	// sample integer choices
	cc := make([]byte, otExtensionCount)
	for i := range cc {
		cc[i] = byte(r.Intn(tuple))
	}

	mLen := make([]int, otExtensionCount)
	for i, m := range nchooseOneMessages {
		mLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	ot, err := NewImprovedKKRT(otExtensionCount, k, tuple, NaorPinkas, crypto.AESCtrDrbg, false, mLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initKKRTReceiver(ot, cc, msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		if err != nil {
			errs <- fmt.Errorf("Error creating improved KKRT OT extension: %s", err)
		}

		ot, err := NewImprovedKKRT(otExtensionCount, k, tuple, NaorPinkas, crypto.AESCtrDrbg, false, mLen)
		if err != nil {
			errs <- err
		}

		err = ot.Send(nchooseOneMessages, conn)
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
	t.Logf("Time taken for improved KKRT OT extension of %d OTs is: %v\n", otExtensionCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("improved KKRT OT extension failed, did not receive any messages")
	}

	for i, m := range msg {
		if !bytes.Equal(m, nchooseOneMessages[i][cc[i]]) {
			t.Logf("choice[%d]=%d\nmessages=%v\n", i, cc[i], nchooseOneMessages[i])
			t.Fatalf("improved KKRT OT extension at msg %d, failed got: %v, want %v", i, m, nchooseOneMessages[i][cc[i]])
		}
	}
}
