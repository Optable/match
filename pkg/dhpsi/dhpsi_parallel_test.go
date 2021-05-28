package dhpsi

import (
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/optable/match/test/emails"
)

// Test the shuffler
func TestDeriveMultiplyParallelShuffler(t *testing.T) {
	var ws sync.WaitGroup
	// pick a ristretto implementation
	gr := NilRistretto(0)
	// get an io pipe to read results
	rcv, snd := io.Pipe()
	// setup a matchables generator
	common := emails.Common(DHPSITestCommonLen)
	matchables := emails.Mix(common, DHPSITestBodyLen)

	// save test matchables
	var sent [][]byte
	// save the permutations
	var permutations []int64
	// save test encoded ristretto points
	var received [][EncodedLen]byte
	// setup sequence
	ws.Add(2)
	// use a channel to hand off the errors
	errs := make(chan error, 2)
	// test
	go func() {
		// Probably advertiser
		defer ws.Done()
		defer snd.Close()
		// make the encoder
		e, err := NewDeriveMultiplyParallelShuffler(snd, DHPSITestLen, gr)
		if err != nil {
			errs <- err
			return
		}
		mm, pp, err := sender(e, gr, matchables)
		sent = mm
		permutations = pp
		if err != nil {
			errs <- err
		}
	}()
	go func() {
		// Probably publisher
		defer ws.Done()
		defer rcv.Close()
		if ee, err := receiver(rcv, DHPSITestLen); err != nil {
			errs <- err
		} else {
			received = ee
		}
	}()

	ws.Wait()

	// errors?
	select {
	case err := <-errs:
		t.Error(err)
	default:
	}

	// check that we received the amount anticipated
	if len(received) != DHPSITestLen {
		t.Errorf("expected %d matchables, got %d", DHPSITestLen, len(received))
	}

	// check that sequences are permutated as expected
	for k, v := range received {
		trimmed := sent[permutations[k]][:32]
		if !compare(v, trimmed) {
			t.Fatalf("shuffle sequence is broken")
		}
	}
}

func TestMultiplyParallelReader(t *testing.T) {
	var wg sync.WaitGroup
	gr := NilRistretto(0)
	// get an io pipe to read results
	rcv, snd := io.Pipe()
	// setup a matchables generator
	common := emails.Common(DHPSITestCommonLen)
	matchables := emails.Mix(common, DHPSITestBodyLen)

	// save sent encoded ristretto points
	var sent [][EncodedLen]byte
	// save received multiplied ristretto points
	var received [][EncodedLen]byte
	// setup sequence
	wg.Add(2)
	// use a channel to hand off the errors
	errs := make(chan error, 2)
	// test
	// setup a writer
	go func() {
		defer wg.Done()
		w, err := NewWriter(snd, DHPSITestLen)
		if err != nil {
			errs <- err
			return
		}
		for identifier := range matchables {
			var point [EncodedLen]byte
			gr.DeriveMultiply(&point, identifier)
			sent = append(sent, point)
			err := w.Write(point)
			if err != nil {
				errs <- fmt.Errorf("Write: %v", err)
				return
			}
		}
	}()

	// setup a parallel reader
	go func() {
		defer wg.Done()
		r, err := NewMultiplyParallelReader(rcv, gr)
		if err != nil {
			errs <- fmt.Errorf("NewMultiplyParallelReader: %v", err)
			return
		}
		for i := int64(0); i < r.Max(); i++ {
			var point [EncodedLen]byte
			err := r.Read(&point)
			if err != nil {
				errs <- fmt.Errorf("Multiply: %v", err)
				break
			}
			received = append(received, point)
		}
	}()

	wg.Wait()

	// errors?
	select {
	case err := <-errs:
		t.Error(err)
	default:
	}

	// same received and sent lenght?
	if len(sent) != len(received) {
		t.Errorf("sent (%d) and received (%d) len not equal", len(sent), len(received))
	}

	// same data sent and received?
	for k, v := range sent {
		if v != received[k] {
			t.Errorf("sent and received data got shuffled")
		}
	}
}
