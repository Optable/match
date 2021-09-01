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
	"sync"
	"time"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
)

var (
	curveBitSet      = "P256"
	cipherModeBitSet = crypto.XORBlake3
)

type kkrtBitSet struct {
	baseOT ot.OTBitSet // base OT under the hood
	m      int         // number of message tuples
	k      int         // width of base OT binary matrix as well as
	// pseudorandom code output length
	prng *rand.Rand // source of randomness
}

// NewKKRT returns a KKRT OPRF
// m: number of message tuples
// k: width of OT extension binary matrix, must be multiple of 64 to work
//    well with BitSet
// baseOT: select which baseOT to use under the hood
// ristretto: baseOT implemented using ristretto
func NewKKRTBitSet(m, k, baseOT int, ristretto bool) (OPRFBitSet, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = m
	}

	ot, err := ot.NewBaseOTBitSet(baseOT, ristretto, k, curve, baseMsgLen, cipherMode)
	if err != nil {
		return kkrtBitSet{}, err
	}

	return kkrtBitSet{baseOT: ot, m: m, k: k, prng: rand.New(rand.NewSource(time.Now().UnixNano()))}, nil
}

// Send returns the OPRF keys
func (o kkrtBitSet) Send(rw io.ReadWriter) (keys []KeyBitSet, err error) {
	// sample random 16 byte (128 bit) secret key for AES-128
	sk := util.SampleBitSetSlice(o.prng, 128)

	// send the secret key
	if _, err := sk.WriteTo(rw); err != nil {
		return nil, err
	}

	// sample choice bits for baseOT
	s := util.SampleBitSetSlice(o.prng, o.k)

	// act as receiver in baseOT to receive q^j
	q := make([]*bitset.BitSet, o.k)
	if err = o.baseOT.Receive(s, q, rw); err != nil {
		return nil, err
	}

	// transpose q to m x k matrix for easier row operations
	q = util.BitMatrixToBitSets(util.ContiguousTranspose(util.BitSetsToBitMatrix(q)))

	// store oprf keys
	keys = make([]KeyBitSet, len(q))
	for j := range q {
		keys[j] = KeyBitSet{sk: sk, s: s, q: q[j]}
	}

	return
}

// Receive returns the OPRF output on receiver's choice strings using OPRF keys
func (o kkrtBitSet) Receive(choices []*bitset.BitSet, rw io.ReadWriter) (t []*bitset.BitSet, err error) {
	if len(choices) != o.m {
		return nil, ot.ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	sk := bitset.New(128)
	if _, err = sk.ReadFrom(rw); err != nil {
		return nil, err
	}

	// Sample m x k matrix T
	t = util.SampleRandomBitSetMatrix(o.prng, o.m, o.k)
	var wg sync.WaitGroup
	var msg = make(chan *bitset.BitSet)
	var errBus = make(chan error)
	// make m pairs of k bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][]*bitset.BitSet, o.m)
	for i := range baseMsgs {
		wg.Add(1)
		go func(i int, msg chan<- *bitset.BitSet) {
			defer wg.Done()
			msg <- t[i]
			//m, err := util.XorBytes(t[i], crypto.PseudorandomCode(sk, o.k, choices[i]))
			m := t[i].SymmetricDifference(crypto.PseudorandomCodeBitSet(sk, o.k, choices[i]))
			msg <- m
		}(i, msg)

		baseMsgs[i] = make([]*bitset.BitSet, 2)
		baseMsgs[i][0] = <-msg
		baseMsgs[i][1] = <-msg
	}

	// wait for all operation to be done
	go func() {
		wg.Wait()
		close(errBus)
		close(msg)
	}()

	//errors?
	for err := range errBus {
		if err != nil {
			return nil, err
		}
	}

	// act as sender in baseOT to send k columns
	if err = o.baseOT.Send(util.BitMatrixToBitSets3D(util.ContiguousTranspose3D(util.BitSetsToBitMatrix3D(baseMsgs))), rw); err != nil {
		return nil, err
	}
	return
}

// Encode computes and returns OPRF(k, in)
func (o kkrtBitSet) Encode(k KeyBitSet, in *bitset.BitSet) *bitset.BitSet {
	// compute q_i ^ (C(r) & s)
	x := crypto.PseudorandomCodeBitSet(k.sk, o.k, in).Intersection(k.s)

	t := k.q.SymmetricDifference(x)

	return t
}
