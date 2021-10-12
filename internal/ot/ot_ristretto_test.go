package ot

import (
	"bytes"
	"testing"

	gr "github.com/bwesterb/go-ristretto"
)

func TestReadWritePoints(t *testing.T) {
	rw := new(bytes.Buffer)
	r := newRistrettoReader(rw)
	w := newRistrettoWriter(rw)

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
