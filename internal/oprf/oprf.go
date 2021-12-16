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
	crand "crypto/rand"
	"encoding/binary"
	"io"
	"math/rand"
	"runtime"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/ot"
	"github.com/optable/match/internal/util"
	"github.com/zeebo/blake3"
)

const (
	// width of base OT binary matrix as well as the output
	// length of PseudorandomCode (in bits)
	baseOTCount            = aes.BlockSize * 4 * 8
	baseOTCountBitmapWidth = aes.BlockSize * 4
)

// Key contains the relaxed OPRF keys: (C, s), (j, q_j)
// oprfKeys is the received OT extension matrix oprfKeys
// chosen with choice bytes secret.
type Key struct {
	secret   []byte   // secret choice bits
	oprfKeys [][]byte // m x k bit matrice
}

// OPRF implements the oprf struct containing the base OT
// as well as the number of message tuples.
type OPRF struct {
	baseOT ot.OT // base OT under the hood
	m      int   // number of message tuples
}

// NewOPRF returns an OPRF where m specifies the number
// of message tuples being exchanged.
func NewOPRF(m int) *OPRF {
	// send k columns of messages of length k/8 (64 bytes)
	baseMsgLens := make([]int, baseOTCount)
	for i := range baseMsgLens {
		baseMsgLens[i] = baseOTCountBitmapWidth // 64 bytes
	}

	return &OPRF{baseOT: ot.NewNaorPinkas(baseMsgLens), m: m}
}

// Send returns the OPRF keys
func (ext *OPRF) Send(rw io.ReadWriter) (*Key, error) {
	// sample choice bits for baseOT
	choices := make([]byte, baseOTCountBitmapWidth)
	if _, err := rand.Read(choices); err != nil {
		return nil, err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]byte, baseOTCount)
	if err := ext.baseOT.Receive(choices, seeds, rw); err != nil {
		return nil, err
	}

	// receive masked columns oprfMask
	paddedLen := util.PadBitMap(ext.m, baseOTCount)
	oprfMask := make([]byte, paddedLen)
	oprfKeys := make([][]byte, baseOTCount)
	prg := blake3.New()
	for col := range oprfKeys {
		if _, err := io.ReadFull(rw, oprfMask); err != nil {
			return nil, err
		}

		oprfKeys[col] = make([]byte, paddedLen)
		if err := crypto.PseudorandomGenerate(oprfKeys[col], seeds[col], prg); err != nil {
			return nil, err
		}

		// Binary AND of each byte in oprfMask with the test bit
		// if bit is 1, we get whole row oprfMask to XOR with
		// oprfKeys[row] if bit is 0, we get a row of 0s which when
		// XORed with oprfKeys[row] just returns the same row, so
		// no need to do an operation
		if util.IsBitSet(choices, col) {
			util.ConcurrentBitOp(util.Xor, oprfKeys[col], oprfMask)
		}
	}
	runtime.GC()
	oprfKeys = util.ConcurrentTransposeWide(oprfKeys)[:ext.m]

	// store oprf keys
	return &Key{secret: choices, oprfKeys: oprfKeys}, nil
}

// Receive returns the hashes of OPRF encodings of choice strings embedded
// in the cuckoo hash table using OPRF keys
func (ext *OPRF) Receive(choices *cuckoo.Cuckoo, secretKey []byte, rw io.ReadWriter) ([]map[uint64]uint64, error) {
	if int(choices.Len()) != ext.m {
		return nil, ot.ErrBaseCountMissMatch
	}

	// compute code word using PseudorandomCode on choice strings in a separate thread
	aesBlock, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, err
	}
	var pseudorandomChan = make(chan [][]byte)
	go func() {
		defer close(pseudorandomChan)
		bitMapLen := util.Pad(ext.m, baseOTCount)
		pseudorandomEncoding := make([][]byte, bitMapLen)
		i := 0
		for ; i < ext.m; i++ {
			idx := choices.GetBucket(uint64(i))
			item, hIdx := choices.GetItemWithHash(idx)
			pseudorandomEncoding[i] = crypto.PseudorandomCode(aesBlock, item, hIdx)
		}
		// pad matrix to ensure the number of rows is divisible by baseOTCount for transposition
		for ; i < len(pseudorandomEncoding); i++ {
			pseudorandomEncoding[i] = make([]byte, baseOTCountBitmapWidth)
		}
		pseudorandomChan <- util.ConcurrentTransposeTall(pseudorandomEncoding)
	}()

	// sample random OT messages
	baseMsgs, err := sampleRandomOTMessages()
	if err != nil {
		return nil, err
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return nil, err
	}

	// read pseudorandomEncodings
	pseudorandomEncoding := <-pseudorandomChan

	oprfEncodings := make([][]byte, baseOTCount)
	paddedLen := util.PadBitMap(ext.m, baseOTCount)
	oprfMask := make([]byte, paddedLen)
	// oprfMask = G(seeds[1])
	// oprfEncoding = G(seeds[0]) ^ oprfMask ^ pseudorandomEncoding
	prg := blake3.New()
	for col := range pseudorandomEncoding {
		oprfEncodings[col] = make([]byte, paddedLen)
		err = crypto.PseudorandomGenerate(oprfEncodings[col], baseMsgs[col][0], prg)
		if err != nil {
			return nil, err
		}

		err = crypto.PseudorandomGenerate(oprfMask, baseMsgs[col][1], prg)
		if err != nil {
			return nil, err
		}

		util.ConcurrentDoubleBitOp(util.DoubleXor, oprfMask, oprfEncodings[col], pseudorandomEncoding[col])

		// send oprfMask
		if _, err = rw.Write(oprfMask); err != nil {
			return nil, err
		}
	}

	runtime.GC()
	oprfEncodings = util.ConcurrentTransposeWide(oprfEncodings)[:ext.m]

	// Hash and index all local encodings
	// the hash value of the oprfEncodings is the key
	// the index of the corresponding ID in the cuckoo hash table is the value
	encodings := make([]map[uint64]uint64, cuckoo.Nhash)
	for i := range encodings {
		encodings[i] = make(map[uint64]uint64, ext.m)
	}
	hasher := choices.GetHasher()
	// hash local oprf output
	for bIdx := uint64(0); bIdx < uint64(len(oprfEncodings)); bIdx++ {
		// check if it was an empty input
		if idx := choices.GetBucket(bIdx); idx != 0 {
			// insert into proper map
			_, hIdx := choices.GetItemWithHash(idx)
			encodings[hIdx][hasher.Hash64(oprfEncodings[bIdx])] = idx
		}
	}

	return encodings, nil
}

// Encode computes and returns the OPRF encoding of a byte slice using an OPRF Key
func (k Key) Encode(rowIdx uint64, pseudorandomEncoding []byte) {
	util.ConcurrentDoubleBitOp(util.AndXor, pseudorandomEncoding, k.secret, k.oprfKeys[rowIdx])
}

// sampleRandomOTMessage allocates a slice of OTMessage, each OTMessage contains a pair of messages.
// Extra elements are added to each column to be a multiple of 512. Every slice is filled with pseudorandom bytes
// values from a rand reader.
func sampleRandomOTMessages() ([]ot.OTMessage, error) {
	var seed int64
	if err := binary.Read(crand.Reader, binary.LittleEndian, &seed); err != nil {
		return nil, err
	}
	rand.Seed(seed)
	// instantiate matrix
	matrix := make([]ot.OTMessage, baseOTCount)
	for row := range matrix {
		for col := range matrix[row] {
			matrix[row][col] = make([]byte, baseOTCountBitmapWidth)
			// fill
			if _, err := rand.Read(matrix[row][col]); err != nil {
				return nil, err
			}
		}
	}

	return matrix, nil
}
