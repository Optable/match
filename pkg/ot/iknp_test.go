package ot

import (
	"math/rand"
	"testing"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))
var s = make([]uint8, 1000)

func BenchmarkSampleBitSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := sampleBitSlice(r, s)
		if err != nil {
			b.Log(err)
		}
	}
}
