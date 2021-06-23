package ot

import (
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestInitCurve(t *testing.T) {
	curveTests := []struct {
		name string
		want string
	}{
		{"P224", "P-224"},
		{"P256", "P-256"},
		{"P384", "P-384"},
		{"P521", "P-521"},
	}

	for _, tt := range curveTests {
		c, _ := initCurve(tt.name)
		got := c.Params().Name
		if got != tt.want {
			t.Fatalf("InitCurve(%s): want curve %s, got curve %s", tt.name, tt.name, got)
		}
	}
}

func TestNewNaorPinkas(t *testing.T) {
	ot, err := NewBaseOt(NaorPinkas, false, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating NaorPinkas baseOt", err)
	}

	if _, ok := ot.(naorPinkas); !ok {
		t.Fatalf("expected type naorPinkas, got %T", ot)
	}
}

func TestNewSimplest(t *testing.T) {
	ot, err := NewBaseOt(Simplest, false, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating Simplest baseOt", err)
	}

	if _, ok := ot.(simplest); !ok {
		t.Fatalf("expected type simplest, got %T", ot)
	}
}

func TestNewUnknownOt(t *testing.T) {
	_, err := NewBaseOt(2, false, 3, curve, []int{1, 2, 3}, cipherMode)
	if err == nil {
		t.Fatal("should get error creating unknown baseOt")
	}
}

func TestNewNaorPinkasRistretto(t *testing.T) {
	ot, err := NewBaseOt(NaorPinkas, true, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating NaorPinkas baseOt", err)
	}

	if _, ok := ot.(naorPinkasRistretto); !ok {
		t.Fatalf("expected type naorPinkasRistretto, got %T", ot)
	}
}

func TestNewSimplestRistretto(t *testing.T) {
	ot, err := NewBaseOt(Simplest, true, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating Simplest baseOt", err)
	}

	if _, ok := ot.(simplestRistretto); !ok {
		t.Fatalf("expected type simplestRistretto, got %T", ot)
	}
}

func TestNewUnknownOtRistretto(t *testing.T) {
	_, err := NewBaseOt(2, true, 3, curve, []int{1, 2, 3}, cipherMode)
	if err == nil {
		t.Fatal("should get error creating unknown baseOt")
	}
}

func TestDeriveKey(t *testing.T) {
	c := elliptic.P256()
	_, px, py, err := elliptic.GenerateKey(c, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	p := elliptic.Marshal(c, px, py)
	key := deriveKey(p)
	if len(key) != 32 {
		t.Fatalf("derived key length is not 32, got: %d", len(key))
	}
}
