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
	"crypto/rand"
	"io"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
)

type imprvKKRT struct {
	baseOT ot.OT // base OT under the hood
	m      int   // number of message tuples
	drbg   int
}

// NewKKRT returns a KKRT OPRF
// m: number of message tuples
// k: width of OT extension binary matrix
// baseOT: select which baseOT to use under the hood
// ristretto: baseOT implemented using ristretto
func newImprovedKKRT(m, baseOT, drbg int, ristretto bool) (OPRF, error) {
	// send k columns of messages of length k/8 (64 bytes)
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = k / 8 // 64 bytes
	}

	ot, err := ot.NewBaseOT(baseOT, ristretto, k, curve, baseMsgLen, cipherMode)
	if err != nil {
		return imprvKKRT{}, err
	}

	return imprvKKRT{baseOT: ot, m: m, drbg: drbg}, nil
}

// Send returns the OPRF keys
func (ext imprvKKRT) Send(rw io.ReadWriter) (keys Key, err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]byte, 16)
	if _, err = rand.Read(sk); err != nil {
		return Key{}, err
	}

	// send the secret key
	if _, err := rw.Write(sk); err != nil {
		return Key{}, err
	}

	// sample choice bits for baseOT
	s := make([]byte, k/8)
	if _, err = rand.Read(s); err != nil {
		return Key{}, err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]uint8, k)
	if err = ext.baseOT.Receive(s, seeds, rw); err != nil {
		return Key{}, err
	}

	// receive masked columns u
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	u := make([]byte, paddedLen)
	q := make([][]byte, k)
	for row := range q {
		if _, err = io.ReadFull(rw, u); err != nil {
			return Key{}, err
		}

		q[row], err = crypto.PseudorandomGenerate(ext.drbg, seeds[row], paddedLen)
		if err != nil {
			return Key{}, err
		}
		err = util.ConcurrentInPlaceXorBytes(q[row], util.AndByte(util.TestBitSetInByte(s, row), u))
		if err != nil {
			return Key{}, err
		}
	}

	q = util.TransposeByteMatrix(q)[:ext.m]

	// store oprf keys
	aesBlock, err := aes.NewCipher(sk)
	return Key{block: aesBlock, s: s, q: q}, err
}

// Receive returns the OPRF output on receiver's choice strings using OPRF keys
func (ext imprvKKRT) Receive(choices *cuckoo.Cuckoo, rw io.ReadWriter) (t [][]byte, err error) {
	if int(choices.Len()) != ext.m {
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
		d := make([][]byte, ext.m)
		aesBlock, _ := aes.NewCipher(sk)
		for i := 0; i < ext.m; i++ {
			idx, _ := choices.GetBucket(uint64(i))
			src, _ := choices.GetItem(idx)
			d[i] = crypto.PseudorandomCode(aesBlock, src)
		}
		pseudorandomChan <- util.TransposeByteMatrix(d)
	}()

	// sample 2*k x k byte matrix (2*k x k bit matrix)
	seeds, err := util.SampleRandomBitMatrix(rand.Reader, 2*k, k)
	if err != nil {
		return nil, err
	}

	baseMsgs := make([][][]byte, k)
	for j := range baseMsgs {
		baseMsgs[j] = make([][]byte, 2)
		baseMsgs[j][0] = seeds[2*j]
		baseMsgs[j][1] = seeds[2*j+1]
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return nil, err
	}

	d := <-pseudorandomChan

	t = make([][]byte, k)
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	var u = make([]byte, paddedLen)
	// u^i = G(seeds[1])
	// t^i = d^i ^ u^i
	for col := range d {
		t[col], err = crypto.PseudorandomGenerate(ext.drbg, baseMsgs[col][0], paddedLen)
		if err != nil {
			return nil, err
		}

		u, err = crypto.PseudorandomGenerate(ext.drbg, baseMsgs[col][1], paddedLen)
		if err != nil {
			return nil, err
		}

		err = util.ConcurrentInPlaceXorBytes(u, t[col])
		if err != nil {
			return nil, err
		}
		err = util.ConcurrentInPlaceXorBytes(u, d[col])
		if err != nil {
			return nil, err
		}

		// send u
		if _, err = rw.Write(u); err != nil {
			return nil, err
		}
	}

	return util.TransposeByteMatrix(t)[:ext.m], nil
}
