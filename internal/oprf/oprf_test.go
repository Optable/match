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
	baseCount = 5000000
	prng      = rand.New(rand.NewSource(time.Now().UnixNano()))
	choices   = genChoiceString()
)

func genChoiceString() [][]byte {
	choices := make([][]byte, baseCount)
	for i := range choices {
		choices[i] = make([]byte, 64)
		prng.Read(choices[i])
	}
	return choices
}
func initOPRFReceiver(oprf OPRF, choices [][]uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}

		go oprfReceiveHandler(conn, oprf, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func oprfReceiveHandler(conn net.Conn, oprf OPRF, choices [][]uint8, outBus chan<- []byte, errs chan<- error) {
	defer close(outBus)

	out, err := oprf.Receive(choices, conn)
	if err != nil {
		errs <- err
	}

	for _, o := range out {
		outBus <- o
	}
}

func testKKRT(t *testing.T) {
	outBus := make(chan []byte)
	keyBus := make(chan Key)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOPRF, err := NewKKRT(baseCount, 424, ot.Simplest, false)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOPRFReceiver(receiverOPRF, choices, outBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	senderOPRF, err := NewKKRT(baseCount, 424, ot.Simplest, false)
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

		defer close(keyBus)
		keys, err := senderOPRF.Send(conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(outBus)
		}

		for _, key := range keys {
			keyBus <- key
		}
	}()

	// Receive keys
	var keys []Key
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
		enc, err := senderOPRF.Encode(keys[i], choices[i])
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(o, enc) {
			t.Logf("choice[%d]=%v\n", i, choices[i])
			t.Fatalf("KKRT OPRF failed, got: %v, want %v", enc, o)
		}
	}
}

func TestImprovedKKRT(t *testing.T) {
	outBus := make(chan []byte)
	keyBus := make(chan Key)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOPRF, err := NewImprovedKKRT(baseCount, 512, ot.Simplest, false)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOPRFReceiver(receiverOPRF, choices, outBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	senderOPRF, err := NewImprovedKKRT(baseCount, 512, ot.Simplest, false)
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

		defer close(keyBus)
		keys, err := senderOPRF.Send(conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(outBus)
		}

		for _, key := range keys {
			keyBus <- key
		}
	}()

	// Receive keys
	var keys []Key
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
	t.Logf("Time taken for %d imporved KKRT OPRF is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(out) == 0 {
		t.Fatal("Improved KKRT OT failed, did not receive any messages")
	}

	for i, o := range out {
		// encode choice with key
		enc, err := senderOPRF.Encode(keys[i], choices[i])
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(o, enc) {
			t.Logf("choice[%d]=%v\n", i, choices[i])
			t.Fatalf("Improved KKRT OPRF failed, got: %v, want %v", enc, o)
		}
	}
}

func testImprovedKKRTBitSetSend(t *testing.T) {
	outBus := make(chan []byte)
	keyBus := make(chan Key)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOPRF, err := NewImprovedKKRTBitSetSend(baseCount, 424, ot.Simplest, false)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOPRFReceiver(receiverOPRF, choices, outBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	senderOPRF, err := NewImprovedKKRTBitSetSend(baseCount, 424, ot.Simplest, false)
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

		defer close(keyBus)
		keys, err := senderOPRF.Send(conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(outBus)
		}

		for _, key := range keys {
			keyBus <- key
		}
	}()

	// Receive keys
	var keys []Key
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
	t.Logf("Time taken for %d imporved KKRT BitSet Send OPRF is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(out) == 0 {
		t.Fatal("Improved KKRT BitSet Send OT failed, did not receive any messages")
	}

	for i, o := range out {
		// encode choice with key
		enc, err := senderOPRF.Encode(keys[i], choices[i])
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(o, enc) {
			t.Logf("choice[%d]=%v\n", i, choices[i])
			t.Fatalf("Improved KKRT BitSet Send OPRF failed, got: %v, want %v", enc, o)
		}
	}
}

func testImprovedKKRTNonTrans(t *testing.T) {
	outBus := make(chan []byte)
	keyBus := make(chan Key)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOPRF, err := NewImprovedKKRTNonTrans(baseCount, 424, ot.Simplest, false)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOPRFReceiver(receiverOPRF, choices, outBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	senderOPRF, err := NewImprovedKKRTNonTrans(baseCount, 424, ot.Simplest, false)
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

		defer close(keyBus)
		keys, err := senderOPRF.Send(conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(outBus)
		}

		for _, key := range keys {
			keyBus <- key
		}
	}()

	// Receive keys
	var keys []Key
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
	t.Logf("Time taken for %d improved KKRT Non Trans OPRF is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(out) == 0 {
		t.Fatal("Improved KKRT Non Trans OT failed, did not receive any messages")
	}

	for i, o := range out {
		// encode choice with key
		enc, err := senderOPRF.Encode(keys[i], choices[i])
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(o, enc) {
			t.Logf("choice[%d]=%v\n", i, choices[i])
			t.Fatalf("Improved KKRT OPRF failed, got: %v, want %v", enc, o)
		}
	}
}
