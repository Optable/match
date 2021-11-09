package oprf

/*
Improved oblivious pseudorandom function (OPRF)
based on KKRT 1 out of 2 OT extension
from the paper: "Efficient Batched Oblivious PRF with Applications to Private Set Intersection"
by Vladimir Kolesnikov, Ranjit Kumaresan, Mike Rosulek, and Ni Treu in 2016, and
the paper "More Efficient Oblivious Transfer Extensions"
by  Gilad Asharov, Yehuda Lindell, Thomas Schneider, and Michael Zohner
and the paper "Extending oblivious transfers efficiently"
by Yuval Ishai, Joe Kilian, Kobbi Nissim, and Erez Petrank for ot-extension using
short secrets.

References:
- http://dx.doi.org/10.1145/2976749.2978381 (KKRT)
- https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf (IKNP)
- https://dl.acm.org/doi/10.1007/s00145-016-9236-6 (ALSZ)

*/

import (
	"crypto/aes"
	"crypto/rand"
	"io"
	"runtime"

	"github.com/minio/highwayhash"
	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

type imprvKKRT struct {
	baseOT ot.OT // base OT under the hood
	m      int   // number of message tuples
	drbg   int
}

// newImprovedKKRT returns an Improved KKRT OPRF
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
	// sample choice bits for baseOT
	s := make([]byte, k/8)
	if _, err = rand.Read(s); err != nil {
		return keys, err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]byte, k)
	if err = ext.baseOT.Receive(s, seeds, rw); err != nil {
		return keys, err
	}

	// receive masked columns u
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	u := make([]byte, paddedLen)
	q := make([][]byte, k)
	h := blake3.New()
	for col := range q {
		if _, err = io.ReadFull(rw, u); err != nil {
			return keys, err
		}

		q[col] = make([]byte, paddedLen)
		err = crypto.PseudorandomGenerateWithBlake3XOF(q[col], seeds[col], h)
		if err != nil {
			return keys, err
		}
		// Binary AND of each byte in u with the test bit
		// if bit is 1, we get whole row u to XOR with q[row]
		// if bit is 0, we get a row of 0s which when XORed
		// with q[row] just returns the same row so no need to do
		// an operation
		if util.BitSetInByte(s, col) {
			err = util.ConcurrentBitOp(util.Xor, q[col], u)
			if err != nil {
				return Key{}, err
			}
		}
	}
	runtime.GC()
	q = util.TransposeByteMatrix(q)[:ext.m]

	// store oprf keys
	return Key{s: s, q: q}, err
}

// Receive returns the OPRF output on receiver's choice strings using OPRF keys
func (ext imprvKKRT) Receive(choices *cuckoo.Cuckoo, sk, seed []byte, rw io.ReadWriter) (encodings [cuckoo.Nhash]map[uint64]uint64, err error) {
	if int(choices.Len()) != ext.m {
		return encodings, ot.ErrBaseCountMissMatch
	}

	// compute code word using pseudorandom code on choice string r in a separate thread
	var pseudorandomChan = make(chan [][]byte)
	var errChan = make(chan error, 1)
	go func() {
		defer close(errChan)
		d := make([][]byte, ext.m)
		aesBlock, err := aes.NewCipher(sk)
		if err != nil {
			errChan <- err
		}
		hasher, err := highwayhash.New128(seed)
		if err != nil {
			errChan <- err
		}
		for i := 0; i < ext.m; i++ {
			idx, err := choices.GetBucket(uint64(i))
			if err != nil {
				errChan <- err
			}
			item, hIdx := choices.GetItemWithHash(idx)
			d[i], err = crypto.PseudorandomCode(aesBlock, hasher, item, hIdx)
			if err != nil {
				errChan <- err
			}
		}
		pseudorandomChan <- util.TransposeByteMatrix(d)
	}()

	// sample 2*k x k byte matrix (2*k x k bit matrix)
	baseMsgs, err := util.SampleRandom3DBitMatrix(rand.Reader, k, 2, k)
	if err != nil {
		return encodings, err
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return encodings, err
	}

	// read error
	var d [][]byte
	select {
	case err := <-errChan:
		if err != nil {
			return encodings, err
		}
	case d = <-pseudorandomChan:
	}

	t := make([][]byte, k)
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	var u = make([]byte, paddedLen)
	// u^i = G(seeds[1])
	// t^i = d^i ^ u^i
	h := blake3.New()
	for col := range d {
		t[col] = make([]byte, paddedLen)
		err = crypto.PseudorandomGenerateWithBlake3XOF(t[col], baseMsgs[col][0], h)
		if err != nil {
			return encodings, err
		}

		err = crypto.PseudorandomGenerateWithBlake3XOF(u, baseMsgs[col][1], h)
		if err != nil {
			return encodings, err
		}

		err = util.ConcurrentDoubleBitOp(util.DoubleXor, u, t[col], d[col])
		if err != nil {
			return encodings, err
		}

		// send u
		if _, err = rw.Write(u); err != nil {
			return encodings, err
		}
	}

	runtime.GC()
	t = util.TransposeByteMatrix(t)[:ext.m]

	// Hash and index all local encodings
	// the hash value of the oprf encoding is the key
	// the index of the corresponding ID in the cuckoo hash table is the value
	for i := range encodings {
		encodings[i] = make(map[uint64]uint64, ext.m)
	}
	hasher := choices.GetHasher()
	// hash local oprf output
	for bIdx := uint64(0); bIdx < uint64(len(t)); bIdx++ {
		// check if it was an empty input
		if idx, err := choices.GetBucket(bIdx); idx != 0 {
			if err != nil {
				return encodings, err
			}
			// insert into proper map
			_, hIdx := choices.GetItemWithHash(idx)
			encodings[hIdx][hasher.Hash64(t[bIdx])] = idx
		}
	}

	//runtime.GC()
	return encodings, nil
}
