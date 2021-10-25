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
	"crypto/rand"
	"fmt"
	"io"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/cuckoo"
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
func (o kkrt) Send(rw io.ReadWriter) (keys Key, err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]byte, 16)
	if _, err = rand.Read(sk); err != nil {
		return Key{}, nil
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
	// act as receiver in baseOT to receive q^j
	q := make([][]byte, k)
	if err = o.baseOT.Receive(s, q, rw); err != nil {
		return Key{}, err
	}

	q = util.TransposeByteMatrix(q)

	aesBlock, err := aes.NewCipher(sk)

	return Key{block: aesBlock, s: s, q: q}, err
}

// Receive returns the OPRF output on receiver's choice strings using OPRF keys
func (o kkrt) Receive(choices *cuckoo.Cuckoo, rw io.ReadWriter) (encodings [cuckoo.Nhash]map[uint64]uint64, err error) {
	if int(choices.Len()) != o.m {
		return encodings, ot.ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	sk := make([]byte, 16)
	if _, err = io.ReadFull(rw, sk); err != nil {
		return encodings, err
	}

	// compute code word using pseudorandom code on choice stirng r in a separate thread
	var pseudorandomChan = make(chan [][]byte)
	go func() {
		d := make([][]byte, o.m)
		aesBlock, _ := aes.NewCipher(sk)
		for i := 0; i < o.m; i++ {
			idx, _ := choices.GetBucket(uint64(i))
			item, hIdx := choices.GetItemWithHash(idx)
			if item == nil {
				fmt.Errorf("failed to retrieve item #%v", idx)
			}
			d[i] = crypto.PseudorandomCode(aesBlock, append(item, hIdx))
		}
		tr := util.TransposeByteMatrix(d)
		pseudorandomChan <- tr
	}()

	// Sample k x m (padded column-wise to multiple of 8 uint64 (512 bits)) matrix T
	t, err := util.SampleRandomBitMatrix(rand.Reader, k, o.m)
	if err != nil {
		return encodings, err
	}

	d := <-pseudorandomChan

	// make k pairs of m bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][][]byte, k)
	for i := range baseMsgs {

		err = util.ConcurrentInPlaceXorBytes(d[i], t[i])
		if err != nil {
			return encodings, err
		}

		baseMsgs[i] = make([][]byte, 2)
		baseMsgs[i][0] = t[i]
		baseMsgs[i][1] = d[i]
	}

	// act as sender in baseOT to send k columns
	if err = o.baseOT.Send(baseMsgs, rw); err != nil {
		return encodings, err
	}

	t = util.TransposeByteMatrix(t)[:o.m]

	// Hash and index all local encodings
	// the hash value of the oprf encoding is the key
	// the index of the corresponding ID is the value
	for i := range encodings {
		encodings[i] = make(map[uint64]uint64, o.m)
	}
	// hash local oprf output
	for bIdx := uint64(0); bIdx < choices.Len(); bIdx++ {
		// check if it was an empty input
		if idx, _ := choices.GetBucket(bIdx); idx != 0 {
			// insert into proper map
			_, hIdx := choices.GetItemWithHash(idx)
			encodings[hIdx][choices.BucketIndices(t[bIdx])[0]] = idx
		}
	}

	return encodings, nil
}
