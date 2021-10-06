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

func TestIKNP(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOT, err := NewIKNP(baseCount, 512, NaorPinkas, false, msgLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initReceiver(receiverOT, choices, msgBus, errs)
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

		senderOT, err := NewIKNP(baseCount, 512, NaorPinkas, false, msgLen)
		if err != nil {
			errs <- err
		}
		err = senderOT.Send(messages, conn)
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
		bit := util.TestBitSetInByte(choices, i)
		if !bytes.Equal(m, messages[i][bit]) {
			t.Fatalf("ALSZ OT extension failed at meesage %d, got: %v, want %v from %v", i, m, messages[i][bit], messages[i])
		}
	}
}

func TestALSZ(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOT, err := NewALSZ(baseCount, 512, NaorPinkas, crypto.AESCtrDrbg, false, msgLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initReceiver(receiverOT, choices, msgBus, errs)
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

		senderOT, err := NewALSZ(baseCount, 512, NaorPinkas, crypto.AESCtrDrbg, false, msgLen)
		if err != nil {
			errs <- err
		}
		err = senderOT.Send(messages, conn)
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
	t.Logf("Time taken for ALSZ OT extension of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("ALSZ OT extension failed, did not receive any messages")
	}

	for i, m := range msg {
		bit := util.TestBitSetInByte(choices, i)
		if !bytes.Equal(m, messages[i][bit]) {
			t.Fatalf("ALSZ OT extension failed at meesage %d, got: %v, want %v from %v", i, m, messages[i][bit], messages[i])
		}
	}
}
