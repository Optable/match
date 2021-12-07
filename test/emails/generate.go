package emails

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
)

const (
	// Prefix is value to be prepended to each generated email
	Prefix = "e:"
	// HashLen is the number of bytes to generate
	HashLen = 32
)

// Common generates the common matchable identifiers
func Common(n, hashLen int) (common []byte) {
	common = make([]byte, n*hashLen)
	if _, err := rand.Read(common); err != nil {
		log.Fatalf("could not generate %d hashes for the common portion", n)
	}
	return
}

// Mix mixes identifiers from common and n new fresh matchables
func Mix(common []byte, n, hashLen int) <-chan []byte {
	// setup the streams
	c1 := commons(common, hashLen)
	c2 := freshes(n, hashLen)
	return mixes(c1, c2)
}

// commons will write HashLen chunks from b to a channel and then close it
func commons(b []byte, hashLen int) <-chan []byte {
	out := make(chan []byte)
	go func() {
		defer close(out)
		for i := 0; i < len(b)/hashLen; i++ {
			hash := b[i*hashLen : i*hashLen+hashLen]
			out <- hash
		}
	}()
	return out
}

// freshes will write a total number of fresh hashes to a channel and then close it
func freshes(total, hashLen int) <-chan []byte {
	out := make(chan []byte)
	go func() {
		defer close(out)
		for i := 0; i < total; i++ {
			b := make([]byte, hashLen)
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
// write the output to a channel and then close it
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
