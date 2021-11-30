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

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

// width of base OT binary matrix  as well as the ouput
// length of PseudorandomCode (in bits)
const baseOTCount = aes.BlockSize * 4 * 8

// Key contains the relaxed OPRF key: (C, s), (j, q_j)
// Pseudorandom code C is represented by a received OT extension matrix otMatrix
// chosen with secret seed secret.
type Key struct {
	secret   []byte   // secret choice bits
	otMatrix [][]byte // m x k bit matrice
}

// OPRF implements the oprf struct containing the base OT
// as well as the number of message tuples.
type OPRF struct {
	baseOT ot.OT // base OT under the hood
	m      int   // number of message tuples
}

// NewOPRF returns an OPRF where m specifies the number
// of message tuples being exchanged.
func NewOPRF(m int) (*OPRF, error) {
	// send k columns of messages of length k/8 (64 bytes)
	baseMsgLen := make([]int, baseOTCount)
	for i := range baseMsgLen {
		baseMsgLen[i] = baseOTCount / 8 // 64 bytes
	}

	ot, err := ot.NewNaorPinkas(baseMsgLen)
	if err != nil {
		return nil, err
	}

	return &OPRF{baseOT: ot, m: m}, nil
}

// Send returns the OPRF keys
func (ext *OPRF) Send(rw io.ReadWriter) (keys Key, err error) {
	// sample choice bits for baseOT
	s := make([]byte, baseOTCount/8)
	if _, err = rand.Read(s); err != nil {
		return keys, err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]byte, baseOTCount)
	if err = ext.baseOT.Receive(s, seeds, rw); err != nil {
		return keys, err
	}

	// receive masked columns u
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	u := make([]byte, paddedLen)
	q := make([][]byte, baseOTCount)
	h := blake3.New()
	for col := range q {
		if _, err = io.ReadFull(rw, u); err != nil {
			return keys, err
		}

		q[col] = make([]byte, paddedLen)
		err = crypto.PseudorandomGenerate(q[col], seeds[col], h)
		if err != nil {
			return keys, err
		}
		// Binary AND of each byte in u with the test bit
		// if bit is 1, we get whole row u to XOR with q[row]
		// if bit is 0, we get a row of 0s which when XORed
		// with q[row] just returns the same row, so no need to do
		// an operation
		if util.IsBitSet(s, col) {
			err = util.ConcurrentBitOp(util.Xor, q[col], u)
			if err != nil {
				return Key{}, err
			}
		}
	}
	runtime.GC()
	q = util.ConcurrentTransposeWide(q, runtime.GOMAXPROCS(0))[:ext.m]

	// store oprf keys
	return Key{secret: s, otMatrix: q}, err
}

// Receive returns the OPRF output on receiver's choice strings using OPRF keys
func (ext *OPRF) Receive(choices *cuckoo.Cuckoo, sk []byte, rw io.ReadWriter) (encodings [cuckoo.Nhash]map[uint64]uint64, err error) {
	if int(choices.Len()) != ext.m {
		return encodings, ot.ErrBaseCountMissMatch
	}

	// compute code word using PseudorandomCode on choice string r in a separate thread
	var pseudorandomChan = make(chan [][]byte)
	var errChan = make(chan error, 1)
	go func() {
		defer close(errChan)
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
			d[i] = crypto.PseudorandomCode(aesBlock, item, hIdx)
		}
		// pad matrix to ensure the number of rows is divisible by 512 for transposition
		pad := util.PadTill512(len(d))
		for i := 0; i < pad; i++ {
			d = append(d, make([]byte, 64))
		}
		pseudorandomChan <- util.ConcurrentTransposeTall(d)
	}()

	// sample 2*k x k byte matrix (2*k x k bit matrix)
	baseMsgs, err := ot.SampleRandomOTMessages(baseOTCount, baseOTCount)
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

	t := make([][]byte, baseOTCount)
	paddedLen := (ext.m + util.PadTill512(ext.m)) / 8
	var u = make([]byte, paddedLen)
	// u^i = G(seeds[1])
	// t^i = d^i ^ u^i
	h := blake3.New()
	for col := range d {
		t[col] = make([]byte, paddedLen)
		err = crypto.PseudorandomGenerate(t[col], baseMsgs[col][0], h)
		if err != nil {
			return encodings, err
		}

		err = crypto.PseudorandomGenerate(u, baseMsgs[col][1], h)
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
	t = util.ConcurrentTransposeWide(t, runtime.GOMAXPROCS(0))[:ext.m]

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

	return encodings, nil
}

// Encode computes and returns OPRF(k, in)
func (k Key) Encode(j uint64, pseudorandomBytes []byte) error {
	return util.ConcurrentDoubleBitOp(util.AndXor, pseudorandomBytes, k.secret, k.otMatrix[j])
}
