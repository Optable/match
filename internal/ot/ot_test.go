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

const (
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

	// create client, server connections
	senderConn, receiverConn := net.Pipe()

	// sender
	go func() {
		senderOT := NewNaorPinkas(msgLen)
		if err := senderOT.Send(messages, senderConn); err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(msgBus)
		}
	}()

	// receiver
	go func() {
		defer close(msgBus)
		receiverOT := NewNaorPinkas(msgLen)

		msg := make([][]byte, baseCount)
		if err := receiverOT.Receive(choices, msg, receiverConn); err != nil {
			errs <- err
		}

		for _, m := range msg {
			msgBus <- m
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
		bit := util.BitExtract(choices, i)
		if !bytes.Equal(m, messages[i][bit]) {
			t.Fatalf("OT failed got: %s, want %s", m, messages[i][bit])
		}
	}
}
