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
	crand "crypto/rand"
	"encoding/binary"
	"io"
	mrand "math/rand"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
)

var (
	curve      = "P256"
	cipherMode = crypto.XORBlake3
)

type kkrt struct {
	baseOT ot.OT // base OT under the hood
	m      int   // number of message tuples
	k      int   // width of base OT binary matrix as well as
	// pseudorandom code output length
	prng *mrand.Rand // source of randomness
}

// NewKKRT returns a KKRT OPRF
// m: number of message tuples
// k: width of OT extension binary matrix
// baseOT: select which baseOT to use under the hood
// ristretto: baseOT implemented using ristretto
func NewKKRT(m, k, baseOT int, ristretto bool) (OPRF, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = m
	}

	ot, err := ot.NewBaseOT(baseOT, ristretto, k, curve, baseMsgLen, cipherMode)
	if err != nil {
		return kkrt{}, err
	}

	// seed math rand with crypto/rand random number
	var seed int64
	binary.Read(crand.Reader, binary.BigEndian, &seed)
	return kkrt{baseOT: ot, m: m, k: k, prng: mrand.New(mrand.NewSource(seed))}, nil
}

// Send returns the OPRF keys
func (o kkrt) Send(rw io.ReadWriter) (keys []Key, err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]uint8, 16)
	if _, err = crand.Read(sk); err != nil {
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
	q = util.ConcurrentColumnarTranspose(q)

	// store oprf keys
	keys = make([]Key, len(q))
	for j := range q {
		keys[j] = Key{sk: sk, s: s, q: q[j]}
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

	// compute code word using pseudorandom code on choice stirng r in a separate thread
	var pseudorandomChan = make(chan [][]byte)
	go func() {
		d := make([][]byte, o.m)
		aesBlock, _ := aes.NewCipher(sk)
		for i := 0; i < o.m; i++ {
			d[i] = crypto.PseudorandomCode(aesBlock, choices[i])
		}
		pseudorandomChan <- util.ConcurrentColumnarTranspose(d)
	}()

	// Sample k x m matrix T
	t, err = util.SampleRandomBitMatrix(o.prng, o.k, o.m)
	if err != nil {
		return nil, err
	}

	d := <-pseudorandomChan

	// make k pairs of m bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][][]byte, o.k)
	for i := range baseMsgs {
		err = util.InPlaceXorBytes(t[i], d[i])
		if err != nil {
			return nil, err
		}
		baseMsgs[i] = make([][]byte, 2)
		baseMsgs[i][0] = t[i]
		baseMsgs[i][1] = d[i]
	}

	// act as sender in baseOT to send k columns
	if err = o.baseOT.Send(baseMsgs, rw); err != nil {
		return nil, err
	}

	return util.ConcurrentColumnarTranspose(t), nil
}

// Encode computes and returns OPRF(k, in)
func (o kkrt) Encode(k Key, in []byte) (out []byte, err error) {
	// compute q_i ^ (C(r) & s)
	aesBlock, _ := aes.NewCipher(k.sk)
	out, err = util.AndBytes(crypto.PseudorandomCode(aesBlock, in), k.s)
	if err != nil {
		return
	}

	err = util.InPlaceXorBytes(k.q, out)

	return
}
