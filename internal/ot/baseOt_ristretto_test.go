package ot

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestSimplestRistretto(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// Start timer
	start := time.Now()
	ot, err := NewBaseOT(Simplest, true, baseCount, curve, msgLen, cipherMode)
	if err != nil {
		t.Fatalf("Error creating simplest ristretto OT: %s", err)
	}

	addr, err := initReceiver(ot, choices, msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		ss, err := NewBaseOT(Simplest, true, baseCount, curve, msgLen, cipherMode)
		if err != nil {
			errs <- fmt.Errorf("Error creating simplest OT: %s", err)
		}

		err = ss.Send(messages, conn)
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
	t.Logf("Time taken for simplest ristretto OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if !bytes.Equal(m, messages[i][choices[i]]) {
			t.Fatalf("OT failed got: %v, want %v", m, messages[i][choices[i]])
		}
	}
}

func TestNaorPinkasRistretto(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()

	receiverOT, err := NewBaseOT(NaorPinkas, true, baseCount, curve, msgLen, cipherMode)
	if err != nil {
		t.Fatalf("Error creating Naor-Pinkas ristretto OT: %s", err)
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
		senderOT, err := NewBaseOT(NaorPinkas, true, baseCount, curve, msgLen, cipherMode)
		if err != nil {
			errs <- fmt.Errorf("Error creating simplest OT: %s", err)
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
	t.Logf("Time taken for NaorPinkas ristretto OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if !bytes.Equal(m, messages[i][choices[i]]) {
			t.Fatalf("OT failed got: %v, want %v", m, messages[i][choices[i]])
		}
	}
}