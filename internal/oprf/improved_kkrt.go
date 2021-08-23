package oprf

/*
Oblivious pseudorandom function (OPRF)
based on KKRT 1 out of 2 OT extension
from the paper: Efficient Batched Oblivious PRF with Applications to Private Set Intersection
by Vladimir Kolesnikov, Ranjit Kumaresan, Mike Rosulek, and Ni Treu in 2016, and improved by Justin Li.
Reference:	http://dx.doi.org/10.1145/2976749.2978381 (KKRT)

It is effectively KKRT OT, but instead of encrypting and decrypting messages,
Send returns the OPRF Keys
Receive returns the OPRF evaluated on inputs using the key: OPRF(k, r)
*/

import (
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

type imprvKKRT struct {
	baseOT ot.OT // base OT under the hood
	m      int   // number of message tuples
	k      int   // width of base OT binary matrix as well as
	// pseudorandom code output length
	prng *rand.Rand // source of randomness
	g    *blake3.Hasher
}

// NewKKRT returns a KKRT OPRF
// m: number of message tuples
// k: width of OT extension binary matrix
// baseOT: select which baseOT to use under the hood
// ristretto: baseOT implemented using ristretto
func NewImprovedKKRT(m, k, baseOT int, ristretto bool) (OPRF, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k
	}

	ot, err := ot.NewBaseOT(baseOT, ristretto, k, curve, baseMsgLen, cipherMode)
	if err != nil {
		return imprvKKRT{}, err
	}
	g := blake3.New()

	return imprvKKRT{baseOT: ot, m: m, k: k, prng: rand.New(rand.NewSource(time.Now().UnixNano())), g: g}, nil
}

// Send returns the OPRF keys
func (ext imprvKKRT) Send(rw io.ReadWriter) (keys []Key, err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]uint8, 16)
	if _, err = ext.prng.Read(sk); err != nil {
		return nil, err
	}

	// send the secret key
	if _, err := rw.Write(sk); err != nil {
		return nil, err
	}

	// sample choice bits for baseOT
	s := make([]uint8, ext.k)
	if err = util.SampleBitSlice(ext.prng, s); err != nil {
		return nil, err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]uint8, ext.k)
	if err = ext.baseOT.Receive(s, seeds, rw); err != nil {
		return nil, err
	}

	// receive masked columns u
	u := make([]byte, ext.m)
	q := make([][]byte, ext.k)
	for col := range q {
		if _, err = io.ReadFull(rw, u); err != nil {
			return nil, err
		}

		q[col], err = crypto.PseudorandomGeneratorWithBlake3(ext.g, seeds[col], ext.m)
		if err != nil {
			return nil, err
		}

		q[col], _ = util.XorBytes(util.AndByte(s[col], u), q[col])
	}

	q = util.Transpose(q)

	// store oprf keys
	keys = make([]Key, len(q))
	for j := range q {
		keys[j] = Key{sk: sk, s: s, q: q[j]}
	}

	return
}

// Receive returns the OPRF output on receiver's choice strings using OPRF keys
func (ext imprvKKRT) Receive(choices [][]byte, rw io.ReadWriter) (t [][]byte, err error) {
	if len(choices) != ext.m {
		return nil, ot.ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	sk := make([]byte, 16)
	if _, err = io.ReadFull(rw, sk); err != nil {
		return nil, err
	}

	// compute code word using pseudorandom code on choice stirng r
	d := make([][]byte, ext.m)
	for row := range d {
		d[row] = crypto.PseudorandomCode(sk, ext.k, choices[row])
	}

	d = util.Transpose(d)

	// sample k x k bit mtrix
	seeds, err := util.SampleRandomBitMatrix(ext.prng, 2*ext.k, ext.k)
	if err != nil {
		return nil, err
	}

	baseMsgs := make([][][]byte, ext.k)
	for j := range baseMsgs {
		baseMsgs[j] = make([][]byte, 2)
		baseMsgs[j][0] = seeds[2*j]
		baseMsgs[j][1] = seeds[2*j+1]
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return nil, err
	}

	t = make([][]byte, ext.k)
	var u = make([]byte, ext.m)
	// u^i = G(seeds[1])
	// t^i = d^i ^ u^i
	for col := range d {
		t[col], err = crypto.PseudorandomGeneratorWithBlake3(ext.g, baseMsgs[col][0], ext.m)
		if err != nil {
			return nil, err
		}

		u, err = crypto.PseudorandomGeneratorWithBlake3(ext.g, baseMsgs[col][1], ext.m)
		if err != nil {
			return nil, err
		}
		u, _ = util.XorBytes(t[col], u)
		u, _ = util.XorBytes(u, d[col])

		// send u
		if _, err = rw.Write(u); err != nil {
			return nil, err
		}
	}

	return util.Transpose(t), nil
}

// Encode computes and returns OPRF(k, in)
func (o imprvKKRT) Encode(k Key, in []byte) (out []byte, err error) {
	// compute q_i ^ (C(r) & s)
	x, err := util.AndBytes(crypto.PseudorandomCode(k.sk, o.k, in), k.s)
	if err != nil {
		return
	}

	t, err := util.XorBytes(k.q, x)
	if err != nil {
		return
	}

	return t, nil
}
