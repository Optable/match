package ot

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/bits-and-blooms/bitset"
)

func initIKNPReceiverBitSet(ot OTBitSet, choices *bitset.BitSet, msgBus chan<- *bitset.BitSet, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}

		go iknpReceiveHandlerBitSet(conn, ot, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

/*
func initImprovedIKNPReceiverBitSet(ot OT, choices []uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
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
*/
func iknpReceiveHandlerBitSet(conn net.Conn, ot OTBitSet, choices *bitset.BitSet, msgBus chan<- *bitset.BitSet, errs chan<- error) {
	defer close(msgBus)

	msg := make([]*bitset.BitSet, baseCount)
	err := ot.Receive(choices, msg, conn)
	if err != nil {
		errs <- err
	}

	for _, m := range msg {
		msgBus <- m
	}
}

func TestIKNPBitSet(t *testing.T) {
	for i, m := range bitsetMessages {
		msgLen[i] = int(m[0].Len())
	}

	msgBus := make(chan *bitset.BitSet)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOT, err := NewIKNPBitSet(baseCount, 128, Simplest, false, msgLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initIKNPReceiverBitSet(receiverOT, bitsetChoices, msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		if err != nil {
			errs <- fmt.Errorf("Error creating IKNPBitSet OT: %s", err)
		}

		senderOT, err := NewIKNPBitSet(baseCount, 128, Simplest, false, msgLen)
		if err != nil {
			errs <- err
		}
		err = senderOT.Send(bitsetMessages, conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(msgBus)
		}

	}()

	// Receive msg
	var msg []*bitset.BitSet
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
	t.Logf("Time taken for IKNPBitSet OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("IKNPBitSet OT failed, did not receive any messages")
	}

	for i, m := range msg {
		var choice uint8
		if bitsetChoices.Test(uint(i)) {
			choice = 1
		}
		if !m.Equal(bitsetMessages[i][choice]) {
			t.Fatalf("IKNPBitSet OT failed at message %d, got: %s, want %s", i, m, bitsetMessages[i][choice])
		}
	}
}

// TODO not modified yet
/*
func TestImprovedIKNPBitSet(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOT, err := NewImprovedIKNP(baseCount, 128, Simplest, false, msgLen)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initImprovedIKNPReceiver(receiverOT, choices, msgBus, errs)
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

		senderOT, err := NewImprovedIKNP(baseCount, 128, Simplest, false, msgLen)
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
	t.Logf("Time taken for Improved IKNP OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("Improved IKNP OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if !bytes.Equal(m, messages[i][choices[i]]) {
			t.Fatalf("Improved IKNP OT failed at meesage %d, got: %v, want %v from %v", i, m, messages[i][choices[i]], messages[i])
		}
	}
}
*/
