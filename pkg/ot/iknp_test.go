package ot

import (
	"math/rand"
	"testing"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))
var n = 100000

func BenchmarkSampleBitSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := sampleBitSlice(r, n)
		if err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkSampleBitSliceInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sampleBitSliceInt(r, n)
	}
}
