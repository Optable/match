package dhpsi

import (
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/optable/match/internal/permutations"
	"github.com/optable/match/test/emails"
)

const (
	DHPSITestCommonLen = 1000
	DHPSITestBodyLen   = 100000
	DHPSITestLen       = DHPSITestBodyLen + DHPSITestCommonLen
)

type ShufflerEncoder interface {
	Shuffle([]byte) (err error)
	Permutations() permutations.Permutations
}

// returns true if b1 and b2 have the same bytes
func compare(b1 [EncodedLen]byte, b2 []byte) bool {
	if len(b2) != EncodedLen {
		return false
	}
	for k, v := range b1 {
		if v != b2[k] {
			return false
		}
	}
	return true
}

// emulate probably an advertiser
func sender(e ShufflerEncoder, r Ristretto, matchables <-chan []byte) ([][]byte, permutations.Permutations, error) {
	// save test matchables
	var sent [][]byte
	// setup stage 1
	var encoder = e
	// save the permutations
	p := encoder.Permutations()
	for matchable := range matchables {
		sent = append(sent, matchable)
		if err := encoder.Shuffle(matchable); err != nil {
			return sent, p, fmt.Errorf("sender error at Shuffle: %v", err)
		}
	}
	// another write should return ErrUnexpectedPoint
	var b = make([]byte, emails.HashLen)
	if err := encoder.Shuffle(b); err != ErrUnexpectedPoint {
		return sent, p, fmt.Errorf("sender expected ErrUnexpectedPoint and got %v", err)
	}

	return sent, p, nil
}

// emulate probably a publisher
func receiver(r io.Reader, n int64) ([][EncodedLen]byte, error) {
	// save test encoded ristretto points
	var received [][EncodedLen]byte
	reader, err := NewReader(r)
	if err != nil {
		return received, err
	}
	if reader.Max() != n {
		return received, fmt.Errorf("receiver expected size %d got %d", n, reader.Max())
	}
	for i := int64(0); i < n; i++ {
		var p [EncodedLen]byte
		err := reader.Read(&p)
		if err != nil {
			if err != io.EOF {
				return received, fmt.Errorf("receiver error at Read: %v", err)
			}
		}
		// save the output
		received = append(received, p)
	}

	// another read should return io.EOF
	var p [EncodedLen]byte
	if err := reader.Read(&p); err != io.EOF {
		return received, fmt.Errorf("receiver expected io.EOF and got %v", err)
	}
	return received, nil
}

// Test the shuffler
func TestDeriveMultiplyShuffler(t *testing.T) {
	var ws sync.WaitGroup
	// pick a ristretto implementation
	gr := NilRistretto(0)
	// get an io pipe to read results
	rcv, snd := io.Pipe()
	// setup a matchables generator
	common := emails.Common(DHPSITestCommonLen, emails.HashLen)
	matchables := emails.Mix(common, DHPSITestBodyLen, emails.HashLen)

	// save test matchables
	var sent [][]byte
	// save the permutations
	var permutations permutations.Permutations
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
		e, err := NewDeriveMultiplyShuffler(snd, DHPSITestLen, gr)
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
		trimmed := sent[permutations.Shuffle(int64(k))][:32]
		if !compare(v, trimmed) {
			t.Fatalf("shuffle sequence is broken")
		}
	}
}
