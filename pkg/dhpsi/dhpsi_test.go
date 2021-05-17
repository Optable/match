package dhpsi

import (
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/optable/match/test/emails"
)

const (
	DHPSITestCommonLen = 10
	DHPSITestBodyLen   = 90
	DHPSITestLen       = DHPSITestBodyLen + DHPSITestCommonLen
)

type ShufflerEncoder interface {
	Shuffle([]byte) (err error)
	Permutations() []int64
}

// test loopback ristretto just copies data out
// and does no treatment
type NilRistretto int

func (g NilRistretto) DeriveMultiply(matchable []byte) [EncodedLen]byte {
	// return first 32 bytes of matchable
	var out [EncodedLen]byte
	copy(out[:], matchable[:32])
	return out
}
func (g NilRistretto) Multiply(encoded [EncodedLen]byte) [EncodedLen]byte {
	// passthrought
	return encoded
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
func sender(w io.Writer, n int64, r Ristretto, matchables <-chan []byte, direct bool) ([][]byte, []int64, error) {
	// save test matchables
	var sent [][]byte
	// save the permutations
	var p []int64
	var encoder ShufflerEncoder
	// setup stage 1
	switch direct {
	case false:
		if e, err := NewDeriveMultiplyShuffler(w, n, r); err != nil {
			return sent, p, fmt.Errorf("error at NewDeriveMultiplyShuffler: %v", err)
		} else {
			encoder = e
		}
	case true:
		if e, err := NewDeriveMultiplyDirectShuffler(w, n, r); err != nil {
			return sent, p, fmt.Errorf("error at NewDeriveMultiplyDirectShuffler: %v", err)
		} else {
			encoder = e
		}
	}
	p = encoder.Permutations()
	for matchable := range matchables {
		sent = append(sent, matchable)
		if err := encoder.Shuffle(matchable); err != nil {
			return sent, p, fmt.Errorf("sender error at Encode: %v", err)
		}
	}
	// another write should return ErrUnexpectedEncodeByte
	var b = make([]byte, emails.HashLen)
	if err := encoder.Shuffle(b); err != ErrUnexpectedEncodeByte {
		return sent, p, fmt.Errorf("sender expected ErrUnexpectedEncodeByte and got %v", err)
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

func BenchmarkDeriveMultiplyEncoder100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var ws sync.WaitGroup
		// pick a ristretto implementation
		gr := NilRistretto(0)
		// get an io pipe to read results
		rcv, snd := io.Pipe()
		// setup a matchables generator
		common := emails.Common(10000)
		matchables := emails.Mix(common, 90000)
		// setup sequence
		ws.Add(2)
		// test
		go func() {
			// Probably advertiser
			defer ws.Done()
			sender(snd, 100000, gr, matchables, false)
		}()
		go func() {
			// Probably publisher
			defer ws.Done()
			receiver(rcv, 100000)
		}()
		ws.Wait()
	}
}

func BenchmarkDeriveMultiplyDirectEncoder100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var ws sync.WaitGroup
		// pick a ristretto implementation
		gr := NilRistretto(0)
		// get an io pipe to read results
		rcv, snd := io.Pipe()
		// setup a matchables generator
		common := emails.Common(10000)
		matchables := emails.Mix(common, 90000)
		// setup sequence
		ws.Add(2)
		// test
		go func() {
			// Probably advertiser
			defer ws.Done()
			sender(snd, 100000, gr, matchables, true)
		}()
		go func() {
			// Probably publisher
			defer ws.Done()
			receiver(rcv, 100000)
		}()
		ws.Wait()
	}
}

// Test the shuffler
func TestDeriveMultiplyEncoder(t *testing.T) {
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
		if mm, pp, err := sender(snd, DHPSITestLen, gr, matchables, false); err != nil {
			errs <- err
		} else {
			sent = mm
			permutations = pp
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

// test the encoder
func TestEncoder(t *testing.T) {
}

func TestReader(t *testing.T) {

}
