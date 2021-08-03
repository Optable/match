package oprf

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/optable/match/internal/ot"
)

var (
	network   = "tcp"
	address   = "127.0.0.1:"
	baseCount = 1024
	prng      = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func initKKRTReceiver(oprf OPRF, choices [][]uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}

		go kkrtReceiveHandler(conn, oprf, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func kkrtReceiveHandler(conn net.Conn, oprf OPRF, choices [][]uint8, outBus chan<- []byte, errs chan<- error) {
	defer close(outBus)

	out, err := oprf.Receive(choices, conn)
	if err != nil {
		errs <- err
	}

	for _, o := range out {
		outBus <- o
	}
}

func TestKKRT(t *testing.T) {
	// sample choice strings
	choices := make([][]byte, baseCount)
	for i := range choices {
		choices[i] = make([]byte, 64)
		prng.Read(choices[i])
	}

	outBus := make(chan []byte)
	keyBus := make(chan key)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	oprf, err := NewKKRT(baseCount, 128, len(choices[0]), ot.Simplest, false)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initKKRTReceiver(oprf, choices, outBus, errs)
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

		oprf, err := NewKKRT(baseCount, 128, len(choices[0]), ot.Simplest, false)
		if err != nil {
			errs <- err
		}

		defer close(keyBus)
		keys, err := oprf.Send(conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(outBus)
		}

		for _, key := range keys {
			keyBus <- key
		}
	}()

	// Receive keys
	var keys []key
	for key := range keyBus {
		keys = append(keys, key)
	}

	// Receive msg
	var out [][]byte
	for o := range outBus {
		out = append(out, o)
	}

	//errors?
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}

	// stop timer
	end := time.Now()
	t.Logf("Time taken for %d KKRT OPRF is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(out) == 0 {
		t.Fatal("KKRT OT failed, did not receive any messages")
	}

	for i, o := range out {
		// encode choice with key
		enc, err := Encode(keys[i], choices[i])
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(o, enc) {
			t.Logf("choice[%d]=%v\n", i, choices[i])
			t.Fatalf("KKRT OPRF failed, got: %v, want %v", enc, o)
		}
	}
}
