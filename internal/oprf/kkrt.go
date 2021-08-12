package oprf

/*
Oblivious pseudorandom function (OPRF)
based on KKRT 1 out of 2 OT extension
from the paper: Efficient Batched Oblivious PRF with Applications to Private Set Intersection
by Vladimir Kolesnikov, Ranjit Kumaresan, Mike Rosulek, and Ni Treu in 2016.
Reference:	http://dx.doi.org/10.1145/2976749.2978381 (KKRT)

It is effectively KKRT OT, but instead of encrypting and decrypting messages,
Send returns the OPRF Keys
Receive returns the OPRF evaluated on inputs using the key: OPRF(k, r)
*/

import (
	"io"
	"math/rand"
	"time"

	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
)

var (
	curve      = "P256"
	cipherMode = ot.XORBlake3
)

type kkrt struct {
	baseOT ot.OT // base OT under the hood
	m      int   // number of message tuples
	k      int   // width of base OT binary matrix as well as
	// pseudorandom code output length
	n    int        // number of messages in each tuple
	prng *rand.Rand // source of randomness
}

// key contains the relaxed OPRF key: (C, s), (j, q_j)
// the index j is implicit when key is stored into a key slice.
// Pseudorandom code C is represented by sk
type key struct {
	sk []byte // secret key for pseudorandom code
	s  []byte // secret choice bits
	q  []byte // m x k bit matrice
}

// NewKKRT returns a KKRT OPRF
// m: number of message tuples
// k: width of OT extension binary matrix
// n: number of messages in each tuple
// baseOT: select which baseOT to use under the hood
// ristretto: baseOT implemented using ristretto
func NewKKRT(m, k, n, baseOT int, ristretto bool) (OPRF, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = m
	}

	ot, err := ot.NewBaseOT(baseOT, ristretto, k, curve, baseMsgLen, cipherMode)
	if err != nil {
		return kkrt{}, err
	}

	return kkrt{baseOT: ot, m: m, k: k, n: n, prng: rand.New(rand.NewSource(time.Now().UnixNano()))}, nil
}

// Send returns the OPRF keys
func (o kkrt) Send(rw io.ReadWriter) (keys []key, err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]uint8, 16)
	if _, err = o.prng.Read(sk); err != nil {
		return nil, nil
	}

	// send the secret key
	if _, err := rw.Write(sk); err != nil {
		return nil, err
	}

	// sample choice bits for baseOT
	s := make([]uint8, o.k)
	if err = util.SampleBitSlice(o.prng, s); err != nil {
		return nil, err
	}

	// act as receiver in baseOT to receive q^j
	q := make([][]uint8, o.k)
	if err = o.baseOT.Receive(s, q, rw); err != nil {
		return nil, err
	}

	// transpose q to m x k matrix for easier row operations
	q = util.Transpose(q)

	// store oprf keys
	keys = make([]key, len(q))
	for j := range q {
		keys[j] = key{sk: sk, s: s, q: q[j]}
	}

	return
}

// Receive returns the OPRF output on receiver's choice strings using OPRF keys
func (o kkrt) Receive(choices [][]byte, rw io.ReadWriter) (t [][]byte, err error) {
	if len(choices) != o.m {
		return nil, ot.ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	sk := make([]byte, 16)
	if _, err = io.ReadFull(rw, sk); err != nil {
		return nil, err
	}

	// Sample m x k matrix T
	t, err = util.SampleRandomBitMatrix(o.prng, o.m, o.k)
	if err != nil {
		return nil, err
	}

	// make m pairs of k bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][][]byte, o.m)
	for i := range baseMsgs {
		baseMsgs[i] = make([][]byte, 2)
		baseMsgs[i][0] = t[i]
		baseMsgs[i][1], err = util.XorBytes(t[i], ot.PseudorandomCode(sk, o.k, choices[i]))
		if err != nil {
			return nil, err
		}
	}

	// act as sender in baseOT to send k columns
	if err = o.baseOT.Send(util.Transpose3D(baseMsgs), rw); err != nil {
		return nil, err
	}

	return
}

// Encode computes and returns OPRF(k, in)
func (o kkrt) Encode(k key, in []byte) (out []byte, err error) {
	// compute q_i ^ (C(r) & s)
	x, err := util.AndBytes(ot.PseudorandomCode(k.sk, o.k, in), k.s)
	if err != nil {
		return
	}

	t, err := util.XorBytes(k.q, x)
	if err != nil {
		return
	}

	return t, nil
}