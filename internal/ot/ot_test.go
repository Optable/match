package ot

import (
	"testing"
)

func TestNewNaorPinkas(t *testing.T) {
	ot, err := NewBaseOT(NaorPinkas, false, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating NaorPinkas baseOT", err)
	}

	if _, ok := ot.(naorPinkas); !ok {
		t.Fatalf("expected type naorPinkas, got %T", ot)
	}
}

func TestNewSimplest(t *testing.T) {
	ot, err := NewBaseOT(Simplest, false, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating Simplest baseOT", err)
	}

	if _, ok := ot.(simplest); !ok {
		t.Fatalf("expected type simplest, got %T", ot)
	}
}

func TestNewUnknownOT(t *testing.T) {
	_, err := NewBaseOT(2, false, 3, curve, []int{1, 2, 3}, cipherMode)
	if err == nil {
		t.Fatal("should get error creating unknown baseOT")
	}
}

func TestNewNaorPinkasRistretto(t *testing.T) {
	ot, err := NewBaseOT(NaorPinkas, true, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating NaorPinkas baseOT", err)
	}

	if _, ok := ot.(naorPinkasRistretto); !ok {
		t.Fatalf("expected type naorPinkasRistretto, got %T", ot)
	}
}

func TestNewSimplestRistretto(t *testing.T) {
	ot, err := NewBaseOT(Simplest, true, 3, curve, []int{1, 2, 3}, cipherMode)
	if err != nil {
		t.Fatalf("got error %v while creating Simplest baseOT", err)
	}

	if _, ok := ot.(simplestRistretto); !ok {
		t.Fatalf("expected type simplestRistretto, got %T", ot)
	}
}

func TestNewUnknownOTRistretto(t *testing.T) {
	_, err := NewBaseOT(2, true, 3, curve, []int{1, 2, 3}, cipherMode)
	if err == nil {
		t.Fatal("should get error creating unknown baseOT")
	}
}
