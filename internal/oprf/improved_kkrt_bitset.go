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
	"crypto/aes"
	crand "crypto/rand"
	"encoding/binary"
	"io"
	mrand "math/rand"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

type imprvKKRTBitSet struct {
	baseOT ot.OTBitSet // base OT under the hood
	m      int         // number of message tuples
	k      int         // width of base OT binary matrix as well as
	// pseudorandom code output length
	prng *mrand.Rand // source of randomness
	g    *blake3.Hasher
}

// NewKKRT returns a KKRT OPRF
// m: number of message tuples
// k: width of OT extension binary matrix
// baseOT: select which baseOT to use under the hood
// ristretto: baseOT implemented using ristretto
func NewImprovedKKRTBitSet(m, k, baseOT int, ristretto bool) (OPRFBitSet, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k
	}

	ot, err := ot.NewBaseOTBitSet(baseOT, ristretto, k, curve, baseMsgLen, cipherMode)
	if err != nil {
		return imprvKKRTBitSet{}, err
	}
	g := blake3.New()

	// seed math rand with crypto/rand random number
	var seed int64
	binary.Read(crand.Reader, binary.BigEndian, &seed)

	return imprvKKRTBitSet{baseOT: ot, m: m, k: k, prng: mrand.New(mrand.NewSource(seed)), g: g}, nil
}

// Send returns the OPRF keys
func (ext imprvKKRTBitSet) Send(rw io.ReadWriter) (keys []KeyBitSet, err error) {
	// sample random 16 byte (128 bit) secret key for AES-128
	sk := util.SampleBitSetSlice(ext.prng, 128)

	// send the secret key
	if _, err := sk.WriteTo(rw); err != nil {
		return nil, err
	}

	// sample choice bits for baseOT
	s := util.SampleBitSetSlice(ext.prng, ext.k)

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([]*bitset.BitSet, ext.k)
	if err = ext.baseOT.Receive(s, seeds, rw); err != nil {
		return nil, err
	}

	// receive masked columns u
	u := bitset.New(uint(ext.m))
	q := make([]*bitset.BitSet, ext.k)
	for col := range q {
		q[col] = crypto.PseudorandomBitSetGeneratorWithBlake3(ext.g, seeds[col], ext.m)
		if _, err := u.ReadFrom(rw); err != nil {
			return nil, err
		}

		util.InPlaceAndBitSet(s.Test(uint(col)), u)
		q[col].InPlaceSymmetricDifference(u)
	}

	q = util.ConcurrentColumnarBitSetTranspose(q)

	// store oprf keys
	keys = make([]KeyBitSet, len(q))
	for j := range q {
		keys[j] = KeyBitSet{sk: sk, s: s, q: q[j]}
	}

	return
}

// Receive returns the OPRF output on receiver's choice strings using OPRF keys
func (ext imprvKKRTBitSet) Receive(choices []*bitset.BitSet, rw io.ReadWriter) (t []*bitset.BitSet, err error) {
	if len(choices) != ext.m {
		return nil, ot.ErrBaseCountMissMatch
	}

	var bitsetMatrixChan = make(chan []*bitset.BitSet)
	go func() {
		seeds := util.SampleRandomBitSetMatrix(ext.prng, 2*ext.k, ext.k)
		bitsetMatrixChan <- seeds
	}()

	// receive AES-128 secret key
	sk := bitset.New(128)
	if _, err = sk.ReadFrom(rw); err != nil {
		return nil, err
	}

	// compute code word using pseudorandom code on choice string r in a separate thread

	go func() {
		d := make([]*bitset.BitSet, ext.m)
		block, _ := aes.NewCipher(util.BitSetToBytes(sk))
		for i := 0; i < ext.m; i++ {
			d[i] = crypto.PseudorandomCodeBitSet(block, choices[i])
		}
		bitsetMatrixChan <- util.ConcurrentColumnarBitSetTranspose(d)
	}()

	// sample k x k bit matrix
	seeds := <-bitsetMatrixChan
	baseMsgs := make([][]*bitset.BitSet, ext.k)
	for j := range baseMsgs {
		baseMsgs[j] = make([]*bitset.BitSet, 2)
		baseMsgs[j][0] = seeds[2*j]
		baseMsgs[j][1] = seeds[2*j+1]
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return nil, err
	}

	// Receive pseudorandom msg from bitSliceChan
	d := <-bitsetMatrixChan

	t = make([]*bitset.BitSet, ext.k)
	var u = bitset.New(uint(ext.m))
	// u^i = G(seeds[1])
	// t^i = d^i ^ u^i
	for col := range d {
		t[col] = crypto.PseudorandomBitSetGeneratorWithBlake3(ext.g, baseMsgs[col][0], ext.m)
		u = crypto.PseudorandomBitSetGeneratorWithBlake3(ext.g, baseMsgs[col][1], ext.m)
		u.InPlaceSymmetricDifference(t[col])
		u.InPlaceSymmetricDifference(d[col])

		// send u
		if _, err = u.WriteTo(rw); err != nil {
			return nil, err
		}
	}

	return util.ConcurrentColumnarBitSetTranspose(t), nil
}
