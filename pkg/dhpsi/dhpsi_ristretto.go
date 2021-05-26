package dhpsi

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"log"

	gr "github.com/bwesterb/go-ristretto"
	r255 "github.com/gtank/ristretto255"
)

const (
	RistrettoTypeGR = iota
	RistrettoTypeR255
)

type Ristretto interface {
	DeriveMultiply(dst *[EncodedLen]byte, src []byte)
	Multiply(dst *[EncodedLen]byte, src [EncodedLen]byte)
}

type GR struct {
	key *gr.Scalar
}

type R255 struct {
	key *r255.Scalar
}

func NewRistretto(t int) (Ristretto, error) {
	switch t {
	case RistrettoTypeGR:
		var key gr.Scalar
		return &GR{key: key.Rand()}, nil
	case RistrettoTypeR255:
		var key = r255.NewScalar()
		var uniformBytes = make([]byte, 64)
		if _, err := rand.Read(uniformBytes); err != nil {
			log.Fatalf("could not generate uniform bytes to seed r255")
		}
		key.FromUniformBytes(uniformBytes)
		return &R255{key: key}, nil
	default:
		return nil, fmt.Errorf("unsupported ristretto type %d", t)
	}
}

// "github.com/bwesterb/go-ristretto"
func (g *GR) DeriveMultiply(dst *[EncodedLen]byte, src []byte) {
	var p gr.Point
	// derive
	p.DeriveDalek(src)
	// multiply
	var q gr.Point
	q.ScalarMult(&p, g.key)
	q.BytesInto(dst)
}

func (g *GR) Multiply(dst *[EncodedLen]byte, src [EncodedLen]byte) {
	// multiply
	var p gr.Point
	p.SetBytes(&src)
	p.ScalarMult(&p, g.key)
	p.BytesInto(dst)
}

// "github.com/gtank/ristretto255"
func (r *R255) DeriveMultiply(dst *[EncodedLen]byte, src []byte) {
	var p = r255.NewElement()
	// derive
	hash := sha512.Sum512(src)
	p.FromUniformBytes(hash[:])
	// multiply
	p.ScalarMult(r.key, p)
	// return.
	var tmp []byte
	tmp = p.Encode(tmp)
	copy(dst[:], tmp)
}

func (r *R255) Multiply(dst *[EncodedLen]byte, src [EncodedLen]byte) {
	// multiply
	var p = r255.NewElement()
	p.Decode(src[:])
	p.ScalarMult(r.key, p)
	// return.
	var tmp []byte
	tmp = p.Encode(tmp)
	copy(dst[:], tmp)
}
