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
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/binary"
	"io"
	mrand "math/rand"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
)

type kkrtBitSet struct {
	baseOT ot.OTBitSet // base OT under the hood
	m      int         // number of message tuples
	k      int         // width of base OT binary matrix as well as
	// pseudorandom code output length
	prng *mrand.Rand // source of randomness
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

	// seed math rand with crypto/rand random number
	var seed int64
	binary.Read(crand.Reader, binary.BigEndian, &seed)
	return kkrtBitSet{baseOT: ot, m: m, k: k, prng: mrand.New(mrand.NewSource(seed))}, nil
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
	q = util.ConcurrentColumnarBitSetTranspose(q)

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

	var bitMatrixChan = make(chan []*bitset.BitSet)
	go func() {
		// Sample k x m matrix T
		bitMatrixChan <- util.ConcurrentColumnarBitSetTranspose(util.SampleRandomBitSetMatrix(o.prng, o.m, o.k))
	}()

	// receive AES-128 secret key
	sk := bitset.New(128)
	if _, err = sk.ReadFrom(rw); err != nil {
		return nil, err
	}

	// compute code word using pseudorandom code on choice string r in a separate thread
	go func() {
		d := make([]*bitset.BitSet, o.m)
		block, _ := aes.NewCipher(util.BitSetToBytes(sk))
		for i := 0; i < o.m; i++ {
			d[i] = crypto.PseudorandomCodeBitSet(block, choices[i])
		}
		bitMatrixChan <- util.ConcurrentColumnarBitSetTranspose(d)
	}()

	// Receive pseudorandom msg from bitMatrixChan
	t = <-bitMatrixChan
	d := <-bitMatrixChan
	//fmt.Println(len(t), len(d), t[0].Len(), d[0].Len())

	// make k pairs of m bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][]*bitset.BitSet, o.k)
	for i := range baseMsgs {
		baseMsgs[i] = make([]*bitset.BitSet, 2)
		baseMsgs[i][0] = t[i]
		baseMsgs[i][1] = t[i].SymmetricDifference(d[i])
	}

	// act as sender in baseOT to send k columns
	if err = o.baseOT.Send(baseMsgs, rw); err != nil {
		return nil, err
	}

	return util.ConcurrentColumnarBitSetTranspose(t), nil
}

// Encode computes and returns OPRF(k, in)
func (o kkrtBitSet) Encode(key KeyBitSet, block cipher.Block, in *bitset.BitSet) *bitset.BitSet {
	// compute q_i ^ (C(r) & s)
	return key.q.SymmetricDifference(crypto.PseudorandomCodeBitSet(block, in).Intersection(key.s))
}
