package ot

/*
1 out of N OT extension
based on IKNP 1 out of 2 OT extension
from the paper: Improved OT extension for Transferring Short Secrets
by Vladimir Kolesnikov and Ranjit Kumaresan in 2013, and later
from the paper: Efficient Batched Oblivious PRF with Applications to Private Set Intersection
by Vladimir Kolesnikov, Ranjit Kumaresan, Mike Rosulek, and Ni Treu in 2016.
Reference:	https://eprint.iacr.org/2013/491.pdf (Improved IKNP)
			http://dx.doi.org/10.1145/2976749.2978381 (KKRT)
*/

import (
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/util"
)

const (
	kkrtCurve      = "P256"
	kkrtCipherMode = crypto.XORBlake3
)

type kkrt struct {
	baseOT OT
	m      int
	k      int
	n      int
	msgLen []int
	prng   *rand.Rand
}

func NewKKRT(m, k, n, baseOT int, ristretto bool, msgLen []int) (kkrt, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i := range baseMsgLen {
		baseMsgLen[i] = m
	}

	ot, err := NewBaseOT(baseOT, ristretto, k, kkrtCurve, baseMsgLen, kkrtCipherMode)
	if err != nil {
		return kkrt{}, err
	}

	return kkrt{baseOT: ot, m: m, k: k, n: n, msgLen: msgLen, prng: rand.New(rand.NewSource(time.Now().UnixNano()))}, nil
}

func (ext kkrt) Send(messages [][][]byte, rw io.ReadWriter) (err error) {
	// sample random 16 byte secret key for AES-128
	sk := make([]uint8, 16)
	if _, err = ext.prng.Read(sk); err != nil {
		return nil
	}

	// send the secret key
	if _, err := rw.Write(sk); err != nil {
		return err
	}

	// sample choice bits for baseOT
	s := make([]uint8, ext.k)
	if err = util.SampleBitSlice(ext.prng, s); err != nil {
		return err
	}

	// act as receiver in baseOT to receive q^j
	q := make([][]uint8, ext.k)
	if err = ext.baseOT.Receive(s, q, rw); err != nil {
		return err
	}

	// transpose q to m x k matrix for easier row operations
	q = util.Transpose(q)

	var key, x, ciphertext []byte
	// encrypt messages and send them
	for i := range messages {
		// proof of concept, suppose we have n messages, and the choice string is an integer in [1, ..., n]
		for choice, plaintext := range messages[i] {
			// compute q_i ^ (C(r) & s)
			x, _ = util.AndBytes(crypto.PseudorandomCode(sk, ext.k, []byte{byte(choice)}), s)
			key, _ = util.XorBytes(q[i], x)

			ciphertext, err = crypto.Encrypt(kkrtCipherMode, key, uint8(choice), plaintext)
			if err != nil {
				return err
			}

			// write ciphertext to reciever
			if _, err = rw.Write(ciphertext); err != nil {
				return err
			}
		}
	}

	return
}

/*
func (ext kkrt) BitSetSend(messages [][]*bitset.BitSet, rw io.ReadWriter) (err error) {
	// sample random 16 byte secret key for AES-128
	sk := util.SampleBitSetSlice(ext.prng, 16)

	// send the secret key
	if _, err := sk.WriteTo(rw); err != nil {
		return err
	}

	// sample choice bits for baseOT
	s := util.SampleBitSetSlice(ext.prng, ext.k)

	// act as receiver in baseOT to receive q^j
	q := make([]*bitset.BitSet, ext.k)
	if err = ext.baseOT.BitSetReceive(s, q, rw); err != nil {
		return err
	}

	// transpose q to m x k matrix for easier row operations
	w := util.BitSetsToBitMatrix(q)
	w = util.ContiguousTranspose(w)
	q = util.BitMatrixToBitSets(w)

	var x, ciphertext, key []byte
	y := bitset.New(0)
	// encrypt messages and send them
	for i := range messages {
		// proof of concept, suppose we have n messages, and the choice string is an integer in [1, ..., n]
		for choice, plaintext := range messages[i] {
			// compute q_i ^ (C(r) & s)
			x = cipher.PseudorandomCode(util.BitSetToBits(sk), ext.k, []byte{byte(choice)})
			y = util.BitsToBitSet(x)
			y.InPlaceIntersection(s)
			key = util.BitSetToBits(q[i].SymmetricDifference(y))

			ciphertext, err = cipher.Encrypt(kkrtCipherMode, key, uint8(choice), util.BitSetToBits(plaintext))
			if err != nil {
				return fmt.Errorf("error encrypting sender message: %s", err)
			}

			// send ciphertext
			if _, err = util.BitsToBitSet(ciphertext).WriteTo(rw); err != nil {
				return err
			}
		}
	}

	return
}
*/
func (ext kkrt) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	sk := make([]byte, 16)
	if _, err = io.ReadFull(rw, sk); err != nil {
		return err
	}

	t, err := util.SampleRandomBitMatrix(ext.prng, ext.m, ext.k)
	if err != nil {
		return err
	}

	var errBus = make(chan error)
	var msg = make(chan []byte)
	var wg sync.WaitGroup
	// make m pairs of k bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][][]byte, ext.m)
	for i := range baseMsgs {
		wg.Add(1)
		go func(i int, msg chan<- []byte) {
			defer wg.Done()
			msg <- t[i]
			m2, err := util.XorBytes(t[i], crypto.PseudorandomCode(sk, ext.k, []byte{choices[i]}))
			msg <- m2
			if err != nil {
				errBus <- err
			}
		}(i, msg)

		baseMsgs[i] = make([][]byte, 2)
		baseMsgs[i][0] = <-msg
		baseMsgs[i][1] = <-msg
	}

	// wait for all operation to be done
	go func() {
		wg.Wait()
		close(errBus)
		close(msg)
	}()
	//errors?
	for err := range errBus {
		if err != nil {
			return err
		}
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOT.Send(util.Transpose3D(baseMsgs), rw); err != nil {
		return err
	}

	e := make([][]byte, ext.n)
	for i := range choices {
		// compute nb of bytes to be read
		l := crypto.EncryptLen(kkrtCipherMode, ext.msgLen[i])
		// read all msg
		for j := range e {
			e[j] = make([]byte, l)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// decrypt received ciphertext using key (choices[i], t_i)
		messages[i], err = crypto.Decrypt(kkrtCipherMode, t[i], choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("error decrypting sender messages: %s", err)
		}
	}

	return
}

/*
func (ext kkrt) BitSetReceive(choices []uint8, messages []*bitset.BitSet, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// receive AES-128 secret key
	sk := bitset.New(16)
	if _, err = sk.ReadFrom(rw); err != nil {
		return err
	}

	// Sample m x k matrix T
	t := util.SampleRandomBitSetMatrix(ext.prng, ext.m, ext.k)

	// make m pairs of k bytes baseOT messages: {t_i, t_i xor C(choices[i])}
	baseMsgs := make([][]*bitset.BitSet, ext.m)
	for i := range baseMsgs {
		var choice byte
		if choices.Test(uint(i)) {
			choice = 1
		}
		baseMsgs[i] = make([]*bitset.BitSet, 2)
		baseMsgs[i][0] = t[i]
		x := cipher.PseudorandomCode(util.BitSetToBits(sk), ext.k, []byte{choice})
		y := util.BitsToBitSet(x)
		baseMsgs[i][1] = t[i].Intersection(y)
		if err != nil {
			return err
		}
	}

	// act as sender in baseOT to send k columns
	baseBitMsgs := util.BitSetsToBitMatrix3D(baseMsgs)
	trBitMsgs := util.ContiguousTranspose3D(baseBitMsgs)
	trMsgs := util.BitMatrixToBitSets3D(trBitMsgs)
	if err = ext.baseOT.Send(trMsgs, rw); err != nil {
		return err
	}

	e := make([]*bitset.BitSet, ext.n)
	for i := range choices {
		// compute nb of bytes to be read
		l := cipher.EncryptLen(kkrtCipherMode, ext.msgLen[i])
		// read all msg
		for j := range e {
			e[j] = bitset.New(uint(l))
			if _, err = e[j].ReadFrom(rw); err != nil {
				return err
			}
		}

		// decrypt received ciphertext using key (choices[i], t_i)
		messages[i], err = cipher.Decrypt(kkrtCipherMode, t[i], choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("error decrypting sender messages: %s", err)
		}
	}

	return
}
*/
