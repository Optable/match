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
	"runtime"

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
		return keys, err
	}

	// send the secret key
	if _, err := rw.Write(sk); err != nil {
		return Key{}, err
	}

	// sample choice bits for baseOT
	s := make([]byte, k/8)
	if _, err = rand.Read(s); err != nil {
		return keys, err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]uint8, k)
	if err = ext.baseOT.Receive(s, seeds, rw); err != nil {
		return keys, err
	}

	// receive masked columns u
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	u := make([]byte, paddedLen)
	q := make([][]byte, k)
	h := blake3.New()
	for row := range q {
		if _, err = io.ReadFull(rw, u); err != nil {
			return keys, err
		}

		q[row] = make([]byte, paddedLen)
		err = crypto.PseudorandomGenerateWithBlake3XOF(q[row], seeds[row], h)
		if err != nil {
			return keys, err
		}
		h.Reset()
		err = util.ConcurrentInPlaceXorBytes(q[row], util.AndByte(util.TestBitSetInByte(s, row), u))
		if err != nil {
			return keys, err
		}
	}
	runtime.GC()
	q = util.TransposeByteMatrix(q)[:ext.m]

	// store oprf keys
	aesBlock, err := aes.NewCipher(sk)
	return Key{block: aesBlock, s: s, q: q}, err
}

// Receive returns the OPRF output on receiver's choice strings using OPRF keys
func (ext imprvKKRT) Receive(choices *cuckoo.Cuckoo, rw io.ReadWriter) (encodings [cuckoo.Nhash]map[uint64]uint64, err error) {
	if int(choices.Len()) != ext.m {
		return encodings, ot.ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	sk := make([]byte, 16)
	if _, err = io.ReadFull(rw, sk); err != nil {
		return encodings, err
	}

	// compute code word using pseudorandom code on choice string r in a separate thread
	var pseudorandomChan = make(chan [][]byte)
	var errChan = make(chan error, 1)
	go func() {
		d := make([][]byte, ext.m)
		aesBlock, err := aes.NewCipher(sk)
		if err != nil {
			errChan <- err
		}
		for i := 0; i < ext.m; i++ {
			idx, err := choices.GetBucket(uint64(i))
			if err != nil {
				errChan <- err
			}
			item, hIdx := choices.GetItemWithHash(idx)
			d[i] = crypto.PseudorandomCodeWithHashIndex(aesBlock, item, hIdx)
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

		err = util.ConcurrentInPlaceDoubleXorBytes(u, t[col], d[col])
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
