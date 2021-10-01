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
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand"
	"time"

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
		baseMsgLen[i] = (m + util.RowsToPad(m)) / 8
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
	sk := make([]byte, 16)
	if _, err = crand.Read(sk); err != nil {
		return nil, nil
	}

	// send the secret key
	if _, err := rw.Write(sk); err != nil {
		return nil, err
	}

	// sample choice bits for baseOT
	s := make([]byte, o.k/8)
	if _, err = crand.Read(s); err != nil {
		return nil, err
	}
	fmt.Println("secret bits", s)

	// act as receiver in baseOT to receive q^j
	q := make([][]byte, o.k)
	if err = o.baseOT.Receive(s, q, rw); err != nil {
		return nil, err
	}

	// transpose q to m x k matrix for easier row operations
	q = util.TransposeByteMatrix(q)[:o.m]
	// store oprf keys
	// TODO is this the wrong number of keys?
	keys = make([]Key, len(q))
	for j := range q {
		keys[j] = Key{sk: sk, s: s, q: q[j]}
	}

	fmt.Println("q ", q[0])
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
		start := time.Now()
		d := make([][]byte, o.m)
		for i := 0; i < o.m; i++ {
			d[i] = crypto.PseudorandomCode(sk, choices[i])
		}
		fmt.Printf("Compute pseudorandom code on %d messages of %d bits each took: %v\n", o.m, o.k, time.Since(start))
		tran := time.Now()
		tr := util.TransposeByteMatrix(d)
		pseudorandomChan <- tr
		fmt.Printf("Compute transpose took: %v\n", time.Since(tran))
	}()

	// Sample k x m (padded column-wise to multiple of 8 uint64 (512 bits)) matrix T
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

	start := time.Now()
	// act as sender in baseOT to send k columns
	if err = o.baseOT.Send(baseMsgs, rw); err != nil {
		return nil, err
	}

	fmt.Println("T: ", util.TransposeByteMatrix(t)[0])

	fmt.Printf("base OT of %d messages of %d bytes each took: %v\n", len(baseMsgs), len(baseMsgs[0][0]), time.Since(start))
	return util.TransposeByteMatrix(t)[:o.m], nil
}
