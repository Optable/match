package oprf

import (
	"crypto/aes"
	"crypto/cipher"
	"io"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

/*
OPRF interface
*/

const (
	KKRT = iota
)

type OPRF interface {
	Send(rw io.ReadWriter) ([]Key, error)
	Receive(choices [][]uint8, rw io.ReadWriter) ([][]byte, error)
	Encode(k Key, in []byte) (out []byte, err error)
}

type OPRFBitSet interface {
	Send(rw io.ReadWriter) ([]KeyBitSet, error)
	Receive(choices []*bitset.BitSet, rw io.ReadWriter) ([]*bitset.BitSet, error)
}

// Key contains the relaxed OPRF key: (C, s), (j, q_j)
// the index j is implicit when key is stored into a key slice.
// Pseudorandom code C is represented by sk
type Key struct {
	sk []byte // secret key for pseudorandom code
	s  []byte // secret choice bits
	q  []byte // m x k bit matrice
}

type KeyBitSet struct {
	sk *bitset.BitSet // secret key for pseudorandom code
	s  *bitset.BitSet // secret choice bits
	q  *bitset.BitSet // m x k bit matrice
}

func GetAESBlock(key KeyBitSet) cipher.Block {
	block, _ := aes.NewCipher(util.BitSetToBytes(key.sk))
	return block
}

// Encode a bitset using oprf key
func Encode(block cipher.Block, key KeyBitSet, in *bitset.BitSet) *bitset.BitSet {
	// compute q_i ^ (C(r) & s)
	return key.q.SymmetricDifference(crypto.PseudorandomCodeBitSet(block, in).Intersection(key.s))
}
