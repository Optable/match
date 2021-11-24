package ot

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/optable/match/internal/util"
)

var (
	network          = "tcp"
	address          = "127.0.0.1:"
	baseCount        = 512
	otExtensionCount = 1400
)

func genMsg(n, t int) []OTMessage {
	data := make([]OTMessage, n)
	for i := 0; i < n; i++ {
		for j := range data[i] {
			data[i][j] = make([]byte, otExtensionCount)
			rand.Read(data[i][j])
		}
	}

	return data
}

func genChoiceBits(n int) []uint8 {
	choices := make([]uint8, n)
	rand.Read(choices)
	return choices
}

func initReceiver(ot OT, choices []uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("cannot create connection in listen accept: %s", err)
		}

		go receiveHandler(conn, ot, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func receiveHandler(conn net.Conn, ot OT, choices []uint8, msgBus chan<- []byte, errs chan<- error) {
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

func TestNaorPinkas(t *testing.T) {
	messages := genMsg(baseCount, 2)
	msgLen := make([]int, len(messages))
	choices := genChoiceBits(baseCount / 8)

	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()

	ot, err := NewNaorPinkas(msgLen)
	if err != nil {
		t.Fatalf("Error creating NaorPinkas OT: %s", err)
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
		ss, err := NewNaorPinkas(msgLen)
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
	t.Logf("Time taken for NaorPinkas OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		var bit byte
		if util.BitSetInByte(choices, i) {
			bit = 1
		}
		if !bytes.Equal(m, messages[i][bit]) {
			t.Fatalf("OT failed got: %s, want %s", m, messages[i][bit])
		}
	}
}
