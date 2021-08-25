package ot

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/cipher"
	"github.com/optable/match/internal/util"
)

/*
1 out of 2 IKNP OT extension
from the paper: Extending Oblivious Transfers Efficiently
by Yushal Ishai, Joe Kilian, Kobbi Nissim, and Erez Petrank in 2003.
reference: https://www.iacr.org/archive/crypto2003/27290145/27290145.pdf

A possible improvement is to use bitset to store the bit matrices/bit sets.
*/

const (
	iknpCurveBitSet      = "P256"
	iknpCipherModeBitSet = cipher.XORBlake3
)

type iknpBitSet struct {
	baseOT OTBitSet
	m      int
	k      int
	msgLen []int
	prng   *rand.Rand
}

func NewIKNPBitSet(m, k, baseOT int, ristretto bool, msgLen []int) (iknpBitSet, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = m
	}

	ot, err := NewBaseOTBitSet(baseOT, ristretto, k, iknpCurveBitSet, baseMsgLen, iknpCipherModeBitSet)
	if err != nil {
		return iknpBitSet{}, err
	}

	return iknpBitSet{baseOT: ot, m: m, k: k, msgLen: msgLen, prng: rand.New(rand.NewSource(time.Now().UnixNano()))}, nil
}

func (ext iknpBitSet) Send(messages [][]*bitset.BitSet, rw io.ReadWriter) (err error) {
	// sample choice bits for baseOT
	s := util.SampleBitSetSlice(ext.prng, ext.k)

	// act as receiver in baseOT to receive q^j
	q := make([]*bitset.BitSet, ext.k)
	if err = ext.baseOT.Receive(s, q, rw); err != nil {
		return err
	}

	// transpose q to m x k matrix for easier row operations
	q = util.BitMatrixToBitSets(util.ContiguousTranspose(util.BitSetsToBitMatrix(q)))

	//var ciphertext []byte
	// encrypt messages and send them
	for i := range messages {
		for choice, plaintext := range messages[i] {
			key := q[i]
			if choice == 1 {
				key = key.SymmetricDifference(s)
			}

			//fmt.Println("sen", plaintext.DumpAsBits())

			//ciphertext, err = cipher.Encrypt(iknpCipherModeBitSet, util.BitSetToBytes(key), uint8(choice), util.BitSetToBytes(plaintext))
			ciphertext, err := cipher.EncryptBitSet(iknpCipherModeBitSet, key, uint8(choice), plaintext)

			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// send ciphertext
			/*
				if _, err = util.BytesToBitSet(ciphertext).WriteTo(rw); err != nil {
					return err
				}
			*/
			if _, err = ciphertext.WriteTo(rw); err != nil {
				return err
			}
		}
	}

	return
}

func (ext iknpBitSet) Receive(choices *bitset.BitSet, messages []*bitset.BitSet, rw io.ReadWriter) (err error) {
	if int(choices.Len()) < len(messages) || int(choices.Len()) > len(messages)+63 || len(messages) != ext.m {
		return ErrBaseCountMissMatch
	}

	// Sample k x m matrix T
	t := util.SampleRandomBitSetMatrix(ext.prng, ext.k, ext.m)

	// make k pairs of m bytes baseOT messages: {t^j, t^j xor choices}
	baseMsgs := make([][]*bitset.BitSet, ext.k)
	for j := range baseMsgs {
		baseMsgs[j] = make([]*bitset.BitSet, 2)
		baseMsgs[j][0] = t[j]
		baseMsgs[j][1] = t[j].SymmetricDifference(choices)
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(baseMsgs, rw); err != nil {
		return err
	}

	// compute k x m transpose to access columns easier
	t = util.BitMatrixToBitSets(util.ContiguousTranspose(util.BitSetsToBitMatrix(t)))

	e := make([]*bitset.BitSet, 2)
	// TODO couldn't this just be "i := range messages" ?
	for i := range messages {
		// compute # of bytes to be read
		l := uint(cipher.EncryptLen(iknpCipherModeBitSet, ext.msgLen[i]))
		// read both msg
		for j := range e {
			e[j] = bitset.New(l)
			if _, err = e[j].ReadFrom(rw); err != nil {
				return err
			}
		}

		// decrypt received ciphertext using key (choices[i], t_i)
		var choice uint8
		if choices.Test(uint(i)) {
			choice = 1
		}
		//message, err := cipher.Decrypt(iknpCipherModeBitSet, util.BitSetToBytes(t[i]), choice, util.BitSetToBytes(e[choice]))
		messages[i], err = cipher.DecryptBitSet(iknpCipherModeBitSet, t[i], choice, e[choice])
		if err != nil {
			return fmt.Errorf("error decrypting sender messages: %s", err)
		}
		//messages[i] = util.BytesToBitSet(message)

		//fmt.Println("rec", messages[i].DumpAsBits())
	}

	return
}
