package ot

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/util"
)

var (
	network        = "tcp"
	address        = "127.0.0.1:"
	curve          = "P256"
	cipherMode     = XORBlake3
	baseCount      = 1024
	messages       = genMsg(baseCount, 2)
	bitsetMessages = genBitSetMsg(baseCount, 2)
	msgLen         = make([]int, len(messages))
	choices        = genChoiceBits(baseCount)
	bitsetChoices  = genChoiceBitSet(baseCount)
	r              = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func genMsg(n, t int) [][][]byte {
	data := make([][][]byte, n)
	for i := 0; i < n; i++ {
		data[i] = make([][]byte, t)
		for j := range data[i] {
			data[i][j] = make([]byte, 64)
			r.Read(data[i][j])
		}
	}

	return data
}

func genBitSetMsg(n, t int) [][]*bitset.BitSet {
	data := make([][]*bitset.BitSet, n)
	for i := 0; i < n; i++ {
		data[i] = make([]*bitset.BitSet, t)
		for j := range data[i] {
			data[i][j] = util.SampleBitSetSlice(r, 64)
		}
	}

	return data
}

func genChoiceBitSet(n int) *bitset.BitSet {
	return util.SampleBitSetSlice(r, n)
}

func genChoiceBits(n int) []uint8 {
	choices := make([]uint8, n)
	util.SampleBitSlice(r, choices)
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

func initBitSetReceiver(ot naorPinkas, choices []uint8, msgBus chan<- *bitset.BitSet, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("cannot create connection in listen accept: %s", err)
		}

		go receiveBitSetHandler(conn, ot, choices, msgBus, errs)
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

func receiveBitSetHandler(conn net.Conn, ot naorPinkas, choices []uint8, msgBus chan<- *bitset.BitSet, errs chan<- error) {
	defer close(msgBus)

	//msg := make([][]byte, baseCount)
	msg := make([]*bitset.BitSet, baseCount)
	//err := ot.Receive(choices, msg, conn)
	err := ot.Receive(util.BytesToBitSet(choices), msg, conn)
	if err != nil {
		errs <- err
	}

	for _, m := range msg {
		msgBus <- m
	}
}

func testSimplestOT(t *testing.T) {
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
		if !bytes.Equal(m, messages[i][choices[i]]) {
			t.Fatalf("OT failed got: %s, want %s", m, messages[i][choices[i]])
		}
	}
}

func TestNaorPinkasOT(t *testing.T) {
	for i, m := range bitsetMessages {
		msgLen[i] = int(m[0].Len())
	}

	msgBus := make(chan *bitset.BitSet)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()

	//ot, err := NewBaseOT(NaorPinkas, false, baseCount, curve, msgLen, cipherMode)
	ot, err := newNaorPinkas(baseCount, curve, msgLen, cipherMode)
	if err != nil {
		t.Fatalf("Error creating NaorPinkas OT: %s", err)
	}

	addr, err := initBitSetReceiver(ot, util.BitSetToBytes(bitsetChoices), msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		ss, err := newNaorPinkas(baseCount, curve, msgLen, cipherMode)
		if err != nil {
			errs <- fmt.Errorf("Error creating simplest OT: %s", err)
		}

		err = ss.Send(bitsetMessages, conn)
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
	t.Logf("Time taken for NaorPinkas OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		var choice uint8
		if bitsetChoices.Test(uint(i)) {
			choice = 1
		}
		if !m.Equal(bitsetMessages[i][choice]) {
			t.Fatalf("OT failed at message %d, got: %s, want %s", i, m, bitsetMessages[i][choice])
		}
	}
}

func benchmarkSampleBitSlice2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		util.SampleBitSlice(r, choices)
	}
}

func BenchmarkSampleBitSetSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		util.SampleBitSetSlice(r, baseCount)
	}
}
