package ot

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

func initOTExtReceiver(ot OT, choices []uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("cannot create connection in listen accept: %s", err)
		}

		go otExtreceiveHandler(conn, ot, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func otExtreceiveHandler(conn net.Conn, ot OT, choices []uint8, msgBus chan<- []byte, errs chan<- error) {
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
func TestIKNP(t *testing.T) {
	mLen := make([]int, otExtensionCount)
	for i, m := range otExtensionMessages {
		mLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOT, err := NewIKNP(otExtensionCount, 512, NaorPinkas, false, mLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOTExtReceiver(receiverOT, otExtensionChoices, msgBus, errs)
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

		senderOT, err := NewIKNP(otExtensionCount, 512, NaorPinkas, false, mLen)
		if err != nil {
			errs <- err
		}
		err = senderOT.Send(otExtensionMessages, conn)
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
	t.Logf("Time taken for IKNP OT of %d OTs is: %v\n", otExtensionCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("IKNP OT failed, did not receive any messages")
	}

	for i, m := range msg {
		bit := util.TestBitSetInByte(otExtensionChoices, i)
		if !bytes.Equal(m, otExtensionMessages[i][bit]) {
			t.Fatalf("ALSZ OT extension failed at meesage %d, got: %v, want %v", i, m, otExtensionMessages[i][bit])
		}
	}
}

func TestALSZ(t *testing.T) {
	mLen := make([]int, otExtensionCount)
	for i, m := range otExtensionMessages {
		mLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOT, err := NewALSZ(otExtensionCount, 512, NaorPinkas, crypto.AESCtrDrbg, false, mLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOTExtReceiver(receiverOT, otExtensionChoices, msgBus, errs)
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

		senderOT, err := NewALSZ(otExtensionCount, 512, NaorPinkas, crypto.AESCtrDrbg, false, mLen)
		if err != nil {
			errs <- err
		}
		err = senderOT.Send(otExtensionMessages, conn)
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
	t.Logf("Time taken for ALSZ OT extension of %d OTs is: %v\n", otExtensionCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("ALSZ OT extension failed, did not receive any messages")
	}

	for i, m := range msg {
		bit := util.TestBitSetInByte(otExtensionChoices, i)
		if !bytes.Equal(m, otExtensionMessages[i][bit]) {
			t.Fatalf("ALSZ OT extension failed at message %d, got: %v, want %v", i, m, otExtensionMessages[i][bit])
		}
	}
}
