package bpsi

import (
	"github.com/devopsfaith/bloomfilter"
	bf "github.com/devopsfaith/bloomfilter/bloomfilter"
)

func SendAll(identifiers <-chan []byte) {
	bf.New(bloomfilter.EmptyConfig)
}
