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
	"crypto/rand"
	"io"

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
}

// NewKKRT returns a KKRT OPRF
// m: number of message tuples
// k: width of OT extension binary matrix
// baseOT: select which baseOT to use under the hood
// ristretto: baseOT implemented using ristretto
func newKKRT(m, baseOT int, ristretto bool) (OPRF, error) {
	// send k columns of messages of length (m (padded to multiple of 512) / 8) bytes
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = (m + util.PadTill512(m)) / 8
	}

	ot, err := ot.NewBaseOT(baseOT, ristretto, k, curve, baseMsgLen, cipherMode)
	if err != nil {
		return kkrt{}, err
	}

	return kkrt{baseOT: ot, m: m}, nil
}

// Send returns the OPRF keys
func (o kkrt) Send(rw io.ReadWriter) (keys []Key, err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]byte, 16)
	if _, err = rand.Read(sk); err != nil {
		return nil, nil
	}

	// send the secret key
	if _, err := rw.Write(sk); err != nil {
		return nil, err
	}

	// sample choice bits for baseOT
	s := make([]byte, k/8)
	if _, err = rand.Read(s); err != nil {
		return nil, err
	}

	// act as receiver in baseOT to receive q^j
	q := make([][]byte, k)
	if err = o.baseOT.Receive(s, q, rw); err != nil {
		return nil, err
	}

	q = util.TransposeByteMatrix(q)

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
		for i := 0; i < o.m; i++ {
			d[i] = crypto.PseudorandomCodeDense(sk, choices[i])
		}
		tr := util.TransposeByteMatrix(d)
		pseudorandomChan <- tr
	}()

	// Sample k x m (padded column-wise to multiple of 8 uint64 (512 bits)) matrix T
	t, err = util.SampleRandomDenseBitMatrix(rand.Reader, k, o.m)
	if err != nil {
		return nil, err
	}

	d := <-pseudorandomChan

	// make k pairs of m bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][][]byte, k)
	for i := range baseMsgs {

		err = util.InPlaceXorBytes(d[i], t[i])
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

	return util.TransposeByteMatrix(t)[:o.m], nil
}
