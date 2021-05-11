package dhpsi

import (
	"crypto/rand"
	"crypto/sha512"
	"log"

	gr "github.com/bwesterb/go-ristretto"
	r255 "github.com/gtank/ristretto255"
)

const (
	RistrettoTypeGR = iota
	RistrettoTypeR255
)

type Ristretto interface {
	DeriveMultiply([]byte) [EncodedLen]byte
	Multiply([EncodedLen]byte) [EncodedLen]byte
}

type GR struct {
	key *gr.Scalar
}

type R255 struct {
	key *r255.Scalar
}

func NewRistretto(t int) Ristretto {
	switch t {
	case RistrettoTypeGR:
		var key gr.Scalar
		return &GR{key: key.Rand()}
	default:
		var key = r255.NewScalar()
		var uniformBytes = make([]byte, 64)
		if _, err := rand.Read(uniformBytes); err != nil {
			log.Fatalf("could not generate uniform bytes to seed r255")
		}
		key.FromUniformBytes(uniformBytes)
		return &R255{key: key}
	}
}

// "github.com/bwesterb/go-ristretto"
func (g *GR) DeriveMultiply(matchable []byte) [EncodedLen]byte {
	var p gr.Point
	// derive
	p.DeriveDalek(matchable)
	// multiply
	var q gr.Point
	q.ScalarMult(&p, g.key)
	// return
	var out [32]byte
	q.BytesInto(&out)
	return out
}

func (g *GR) Multiply(encoded [EncodedLen]byte) [EncodedLen]byte {
	// multiply
	var p gr.Point
	p.SetBytes(&encoded)
	p.ScalarMult(&p, g.key)
	// return
	var out [32]byte
	p.BytesInto(&out)
	return out
}

// "github.com/gtank/ristretto255"
func (r *R255) DeriveMultiply(matchable []byte) [EncodedLen]byte {
	var p = r255.NewElement()
	// derive
	hash := sha512.Sum512(matchable)
	p.FromUniformBytes(hash[:])
	// multiply
	p.ScalarMult(r.key, p)
	// return
	var out [32]byte
	p.Encode(out[:])
	return out
}

func (r *R255) Multiply(encoded [EncodedLen]byte) [EncodedLen]byte {
	// multiply
	var p = r255.NewElement()
	p.Decode(encoded[:])
	p.ScalarMult(r.key, p)
	// return
	var out [32]byte
	p.Encode(out[:])
	return out
}
