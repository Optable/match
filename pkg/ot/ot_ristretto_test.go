package ot

import (
	"bytes"
	"testing"

	gr "github.com/bwesterb/go-ristretto"
)

func TestReadWritePoints(t *testing.T) {
	rw := new(bytes.Buffer)
	r := newReaderRistretto(rw)
	w := newWriterRistretto(rw)

	var point, readPoint gr.Point
	point.Rand()
	readPoint.SetZero()

	if point.Equals(&readPoint) {
		t.Fatal("Read point should not be equal to point")
	}

	w.write(&point)
	r.read(&readPoint)

	if !point.Equals(&readPoint) {
		t.Fatalf("Read point is not the same as the written point, want: %v, got: %v", point.Bytes(), readPoint.Bytes())
	}
}

func TestNewNaorPinkasRistretto(t *testing.T) {
	ot, err := NewBaseOtRistretto(0, 3, []int{1, 2, 3})
	if err != nil {
		t.Fatalf("got error %v while creating NaorPinkas baseOt", err)
	}

	if _, ok := ot.(naorPinkasRistretto); !ok {
		t.Fatalf("expected type naorPinkasRistretto, got %T", ot)
	}
}

func TestNewSimplestRistretto(t *testing.T) {
	ot, err := NewBaseOtRistretto(1, 3, []int{1, 2, 3})
	if err != nil {
		t.Fatalf("got error %v while creating Simplest baseOt", err)
	}

	if _, ok := ot.(simplestRistretto); !ok {
		t.Fatalf("expected type simplestRistretto, got %T", ot)
	}
}

func TestNewUnknownOtRistretto(t *testing.T) {
	_, err := NewBaseOtRistretto(2, 3, []int{1, 2, 3})
	if err == nil {
		t.Fatal("should get error creating unknown baseOt")
	}
}

func TestGenerateKeys(t *testing.T) {
	s, P := generateKeys()
	// check point
	var pP gr.Point
	pP.ScalarMultBase(&s)
	if !P.Equals(&pP) {
		t.Fatal("error in generateKey(), secret, public key pairs not working.")
	}
}

func TestDeriveKeyRistretto(t *testing.T) {
	var p gr.Point
	p.Rand()
	key, err := deriveKeyRistretto(&p)
	if err != nil {
		t.Fatal(err)
	}

	if len(key) != 32 {
		t.Fatalf("derived key length is not 32, got: %d", len(key))
	}
}
