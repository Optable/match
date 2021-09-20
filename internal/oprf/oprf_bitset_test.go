package oprf

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
)

var (
	choicesBitSet = util.SampleRandomBitSetMatrix(prng, baseCount, 64)
	// the following are for encode benchmark
	encoderOPRF, _ = NewImprovedKKRTBitSet(baseCount, 512, ot.Simplest, false)
	sk             = util.SampleBitSetSlice(prng, baseCount)
	s              = util.SampleBitSetSlice(prng, baseCount)
	q              = util.SampleBitSetSlice(prng, baseCount)
	key            = KeyBitSet{sk: sk, s: s, q: q}
	in             = util.SampleBitSetSlice(prng, baseCount)
)

func initOPRFReceiverBitSet(oprf OPRFBitSet, choices []*bitset.BitSet, msgBus chan<- *bitset.BitSet, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}

		go oprfReceiveHandlerBitSet(conn, oprf, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func oprfReceiveHandlerBitSet(conn net.Conn, oprf OPRFBitSet, choices []*bitset.BitSet, outBus chan<- *bitset.BitSet, errs chan<- error) {
	defer close(outBus)

	out, err := oprf.Receive(choices, conn)
	if err != nil {
		errs <- err
	}

	for _, o := range out {
		outBus <- o
	}
}

func TestKKRTBitSet(t *testing.T) {
	outBus := make(chan *bitset.BitSet)
	keyBus := make(chan KeyBitSet)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOPRF, err := NewKKRTBitSet(baseCount, k, ot.Simplest, false)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOPRFReceiverBitSet(receiverOPRF, choicesBitSet, outBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	senderOPRF, err := NewKKRTBitSet(baseCount, k, ot.Simplest, false)
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
	var keys []KeyBitSet
	for key := range keyBus {
		keys = append(keys, key)
	}
	// Receive msg
	var out []*bitset.BitSet
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
	t.Logf("Time taken for %d KKRT BitSet OPRF is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(out) == 0 {
		t.Fatal("KKRT OT failed, did not receive any messages")
	}

	aesBlock := GetAESBlock(keys[0])
	for i, o := range out {
		// encode choice with key
		enc := Encode(aesBlock, keys[i], choicesBitSet[i])

		if !o.Equal(enc) {
			t.Logf("choice[%d]=%v\n", i, choicesBitSet[i])
			t.Fatalf("KKRT OPRF failed, got: %v, want %v", enc, o)
		}
	}
}

func TestImprovedKKRTBitSet(t *testing.T) {
	outBus := make(chan *bitset.BitSet)
	keyBus := make(chan KeyBitSet)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	receiverOPRF, err := NewImprovedKKRTBitSet(baseCount, k, ot.Simplest, false)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOPRFReceiverBitSet(receiverOPRF, choicesBitSet, outBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	senderOPRF, err := NewImprovedKKRTBitSet(baseCount, k, ot.Simplest, false)
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
	var keys []KeyBitSet
	for key := range keyBus {
		keys = append(keys, key)
	}

	// Receive msg
	var out []*bitset.BitSet
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
	t.Logf("Time taken for %d improved KKRT BitSet OPRF is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(out) == 0 {
		t.Fatal("Improved KKRT OT failed, did not receive any messages")
	}

	aesBlock := GetAESBlock(keys[0])
	for i, o := range out[:baseCount] {
		// encode choice with key
		enc := Encode(aesBlock, keys[i], choicesBitSet[i])

		if !o.Equal(enc) {
			t.Logf("choice[%d]=%v\n", i, choicesBitSet[i])
			t.Fatalf("Improved KKRT OPRF failed, got: %v, want %v", enc, o)
		}
	}
}

func BenchmarkImprvKKRTEncode(t *testing.B) {
	aesBlock := GetAESBlock(key)
	for i := 0; i < t.N; i++ {
		fmt.Println(i)
		Encode(aesBlock, key, in)
	}
}
