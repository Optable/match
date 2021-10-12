package ot

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

func TestNewNaorPinkas(t *testing.T) {
	ot, err := NewBaseOT(NaorPinkas, false, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating NaorPinkas baseOT", err)
	}

	if _, ok := ot.(naorPinkas); !ok {
		t.Fatalf("expected type naorPinkas, got %T", ot)
	}
}

func TestNewSimplest(t *testing.T) {
	ot, err := NewBaseOT(Simplest, false, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating Simplest baseOT", err)
	}

	if _, ok := ot.(simplest); !ok {
		t.Fatalf("expected type simplest, got %T", ot)
	}
}

func TestNewUnknownOT(t *testing.T) {
	_, err := NewBaseOT(2, false, 3, curve, []int{1, 2, 3}, cipherMode)
	if err == nil {
		t.Fatal("should get error creating unknown baseOT")
	}
}

func TestNewNaorPinkasRistretto(t *testing.T) {
	ot, err := NewBaseOT(NaorPinkas, true, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating NaorPinkas baseOT", err)
	}

	if _, ok := ot.(naorPinkasRistretto); !ok {
		t.Fatalf("expected type naorPinkasRistretto, got %T", ot)
	}
}

func TestNewSimplestRistretto(t *testing.T) {
	ot, err := NewBaseOT(Simplest, true, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating Simplest baseOT", err)
	}

	if _, ok := ot.(simplestRistretto); !ok {
		t.Fatalf("expected type simplestRistretto, got %T", ot)
	}
}

func TestNewUnknownOTRistretto(t *testing.T) {
	_, err := NewBaseOT(2, true, 3, curve, []int{1, 2, 3}, cipherMode)
	if err == nil {
		t.Fatal("should get error creating unknown baseOT")
	}
}

var (
	network             = "tcp"
	address             = "127.0.0.1:"
	curve               = "P256"
	cipherMode          = crypto.XORBlake3
	baseCount           = 512
	otExtensionCount    = 1400
	tuple               = 10
	messages            = genMsg(baseCount, 2)
	otExtensionMessages = genMsg(otExtensionCount, 2)
	nchooseOneMessages  = genMsg(otExtensionCount, tuple)
	msgLen              = make([]int, len(messages))
	choices             = genChoiceBits(baseCount / 8)
	otExtensionChoices  = genChoiceBits(otExtensionCount / 8)
	r                   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func genMsg(n, t int) [][][]byte {
	data := make([][][]byte, n)
	for i := 0; i < n; i++ {
		data[i] = make([][]byte, t)
		for j := range data[i] {
			data[i][j] = make([]byte, otExtensionCount)
			r.Read(data[i][j])
		}
	}

	return data
}

func genChoiceBits(n int) []uint8 {
	choices := make([]uint8, n)
	r.Read(choices)
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

func TestSimplest(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()

	receiverOT, err := NewBaseOT(Simplest, false, baseCount, curve, msgLen, cipherMode)
	if err != nil {
		t.Fatalf("Error creating Simplest OT: %s", err)
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
		senderOT, err := NewBaseOT(Simplest, false, baseCount, curve, msgLen, cipherMode)
		if err != nil {
			errs <- fmt.Errorf("Error creating simplest OT: %s", err)
		}

		err = senderOT.Send(messages, conn)
		if err != nil {
			errs <- fmt.Errorf("send encountered error: %s", err)
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
	t.Logf("Time taken for simplest OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		bit := util.TestBitSetInByte(choices, i)
		if !bytes.Equal(m, messages[i][bit]) {
			t.Fatalf("OT failed got: %s, want %s", m, messages[i][choices[i]])
		}
	}
}

func TestNaorPinkas(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()

	ot, err := NewBaseOT(NaorPinkas, false, baseCount, curve, msgLen, cipherMode)
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
		ss, err := NewBaseOT(NaorPinkas, false, baseCount, curve, msgLen, cipherMode)
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
		bit := util.TestBitSetInByte(choices, i)
		if !bytes.Equal(m, messages[i][bit]) {
			t.Fatalf("OT failed got: %s, want %s", m, messages[i][bit])
		}
	}
}

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
		bit := util.TestBitSetInByte(choices, i)
		if !bytes.Equal(m, messages[i][bit]) {
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
		bit := util.TestBitSetInByte(choices, i)
		if !bytes.Equal(m, messages[i][bit]) {
			t.Fatalf("OT failed got: %v, want %v", m, messages[i][choices[i]])
		}
	}
}
