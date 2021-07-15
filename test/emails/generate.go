package emails

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
)

const (
	Prefix  = "e:"
	HashLen = 32
)

// Common generates the common segment
func Common(n int) (common []byte) {
	common = make([]byte, n*HashLen)
	if _, err := rand.Read(common); err != nil {
		log.Fatalf("could not generate %d hashes for the common portion", n)
	}
	return
}

// Mix in from common and add n new fresh matchables
func Mix(common []byte, n int) <-chan []byte {
	// setup the streams
	c1 := commons(common)
	c2 := freshes(n)
	return mixes(c1, c2)
}

// commons will write HashLen chunks from b to a channel and then close it
func commons(b []byte) <-chan []byte {
	out := make(chan []byte)
	go func() {
		defer close(out)
		for i := 0; i < len(b)/HashLen; i++ {
			hash := b[i*HashLen : i*HashLen+HashLen]
			out <- hash
		}
	}()
	return out
}

// freshes will write a total number of fresh hashes to a channel and then close it
func freshes(total int) <-chan []byte {
	out := make(chan []byte)
	go func() {
		defer close(out)
		for i := 0; i < total; i++ {
			b := make([]byte, HashLen)
			if _, err := rand.Read(b); err == nil {
				out <- b
			}
		}
	}()
	return out
}

// prefix a byte value with the local preset prefix
func prefix(value []byte) []byte {
	// make final string
	out := make([]byte, len(Prefix)+hex.EncodedLen(len(value)))
	// copy the prefix first and then the
	// hex string
	copy(out, Prefix)
	hex.Encode(out[len(Prefix):], value)
	//  and return this
	//return append(out, "\r\n"...)
	return out
}

// mixes will read c1 & c2 to exhaustion, add the prefix,
// write the output a channel and then close it
func mixes(c1, c2 <-chan []byte) <-chan []byte {
	var ws sync.WaitGroup
	out := make(chan []byte)
	// fixed to 2 here because this is the pattern
	ws.Add(2)
	// exhaust a channel
	f := func(c <-chan []byte) {
		defer ws.Done()
		for b := range c {
			b = prefix(b)
			out <- b
		}
	}
	// fan in c1 & c2
	go f(c1)
	go f(c2)
	// and wait so we can close the out channel
	go func() {
		ws.Wait()
		close(out)
	}()

	return out
}
