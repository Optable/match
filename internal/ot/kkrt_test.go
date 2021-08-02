package ot

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/optable/match/internal/util"
)

var prng = rand.New(rand.NewSource(time.Now().UnixNano()))

func TestTranspose3D(t *testing.T) {
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([][2][]byte, 4)
	for m := range b {
		b[m][0] = make([]byte, 8)
		b[m][1] = make([]byte, 8)
		util.SampleBitSlice(prng, b[m][0])
		util.SampleBitSlice(prng, b[m][1])
	}

	for m := range b {
		if !bytes.Equal(b[m][0], util.Transpose3D(util.Transpose3D(b))[m][0]) {
			t.Fatalf("Transpose of transpose should be equal")
		}

		if !bytes.Equal(b[m][1], util.Transpose3D(util.Transpose3D(b))[m][1]) {
			t.Fatalf("Transpose of transpose should be equal")
		}
	}
}

func TestTranspose(t *testing.T) {

	b := make([][]byte, 4)
	for m := range b {
		b[m] = make([]byte, 8)
		util.SampleBitSlice(prng, b[m])
	}

	for m := range b {
		if !bytes.Equal(b[m], util.Transpose(util.Transpose(b))[m]) {
			t.Fatalf("Transpose of transpose should be equal")
		}
	}
}
