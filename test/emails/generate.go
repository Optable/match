package emails

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
)

// from fake_hashes.sh:
// 	Usage: fake_hashes.sh n_hashes length prefix
//
// generate n_hashes of length length and prefix them with prefix
//
// example:
//  e:0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
//  e:59245d7c68b28404e068b15cba430082549b845ab412c4c3b31fb8632fd794e1
//  e:8d4acbaaec5a4b00465fa6db04deeb7de8722ef2893a3c22096fafe060686c38
//  e:63bc6f65441e7650fd5d3add5f16029bd44abb6af16eeeb532ddfd85865706b5
//  e:d427b4d8dfcfcd4e972d3f836da9457ac0e2dce26f46d9efd264d7af2440b892
//  e:e33da25066c2d7d959de91844f593ac6a8991829f4fda71a2aefb9445745cca1
//  e:6a534d2073cf17f6f42784934dd3b6f3776a9ebd28a2a1594c0f5e0ff3d58002
//  e:fc7f9988396ef0144486c4c7e108f8d3833024d6931e4dc554fc9b1f581f5260
//
// hashes are random blobs of length length expressed in hex and prefixed with a string

// FIX THIS TO RETURNS EMAILS! with the prefix

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

// Prefix a byte value with the local preset prefix
// and add \r\n at the end
func prefix(value []byte) []byte {
	// make final string
	out := make([]byte, len(Prefix)+hex.EncodedLen(len(value)))
	// copy the prefix first and then the
	// hex string
	copy(out, Prefix)
	hex.Encode(out[len(Prefix):], value)
	//  and return this
	return append(out, "\r\n"...)
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
