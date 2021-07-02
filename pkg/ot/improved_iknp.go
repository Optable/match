package ot

import (
	"fmt"
	//"golang.org/x/crypto/sha3"
	"io"
	"math/rand"
	"time"

	"github.com/zeebo/blake3"
)

type imprvIknp struct {
	baseOt Ot
	m      int
	k      int
	msgLen []int
	prng   *rand.Rand
	g      *blake3.Hasher
}

func NewImprovedIknp(m, k, baseOt int, ristretto bool, msgLen []int) (imprvIknp, error) {
	// send k columns of messages of length k
	baseMsgLen := make([]int, k)
	for i, _ := range baseMsgLen {
		baseMsgLen[i] = k
	}

	ot, err := NewBaseOt(baseOt, ristretto, k, iknpCurve, baseMsgLen, iknpCipherMode)
	if err != nil {
		return imprvIknp{}, err
	}

	return imprvIknp{baseOt: ot, m: m, k: k, msgLen: msgLen,
		prng: rand.New(rand.NewSource(time.Now().UnixNano())),
		g:    blake3.New()}, nil
}

func (ext imprvIknp) Send(messages [][2][]byte, rw io.ReadWriter) (err error) {
	// sample choice bits for baseOT
	s := make([]uint8, ext.k)
	if err = sampleBitSlice(ext.prng, s); err != nil {
		return err
	}

	// act as receiver in baseOT to receive k x k seeds for the pseudorandom generator
	seeds := make([][]uint8, ext.k)
	if err = ext.baseOt.Receive(s, seeds, rw); err != nil {
		return err
	}

	// receive masked columns
	q := make([][]byte, ext.k)
	e := make([][]byte, 2)
	for i := range q {
		// read both msg
		for j, _ := range e {
			// each column is m bytes long
			e[j] = make([]byte, ext.m)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}
		// unmask e and store it in q.
		//fmt.Printf("seeds[i]:\n%v\n, e[s[i]]:\n%v\n", seeds[i], e[s[i]])
		q[i], err = xorCipherWithBlake3(seeds[i], s[i], e[s[i]])
		//q[i], err = xorCipherWithPRG(ext.g, seeds[i], e[s[i]])
	}

	//fmt.Printf("q:\n%v\n", q)
	q = transpose(q)

	// encrypt messages and send them
	var key, ciphertext []byte
	for i := range messages {
		for choice, plaintext := range messages[i] {
			key = q[i]
			if choice == 1 {
				key, err = xorBytes(q[i], s)
				if err != nil {
					return err
				}
			}

			ciphertext, err = encrypt(iknpCipherMode, key, uint8(choice), plaintext)
			if err != nil {
				return fmt.Errorf("Error encrypting sender message: %s\n", err)
			}

			// send ciphertext
			if _, err = rw.Write(ciphertext); err != nil {
				return err
			}
		}
	}

	return
}

func (ext imprvIknp) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// compute actual messages to be sent
	// t is pseudorandom binary matrix
	t, err := sampleRandomBitMatrix(ext.prng, ext.k, ext.m)
	if err != nil {
		return err
	}

	// make k pairs of m bytes baseOT messages: {t^j, t^j xor choices}
	baseMsgs := make([][2][]byte, ext.k)
	for j := range baseMsgs {
		for b := range baseMsgs[j] {
			baseMsgs[j][b] = make([]byte, ext.k)
			err = sampleBitSlice(ext.prng, baseMsgs[j][b])
			if err != nil {
				return err
			}
		}
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOt.Send(baseMsgs, rw); err != nil {
		return err
	}

	// compute actual t^j, u^j and send them masked: t^j xor G(s)
	// u^j = t^j xor choice
	var u []uint8
	//u := make([][]uint8, ext.k)
	for row := range t {
		//u, err = xorBytes(t[row], choices)
		u, err = xorBytes(t[row], choices)
		if err != nil {
			return err
		}

		maskedTj, err := xorCipherWithBlake3(baseMsgs[row][0], 0, t[row])
		//maskedTj, err := xorCipherWithPRG(ext.g, baseMsgs[row][0], t[row])
		if err != nil {
			return err
		}

		maskedUj, err := xorCipherWithBlake3(baseMsgs[row][1], 1, u)
		//maskedUj, err := xorCipherWithPRG(ext.g, baseMsgs[row][1], u)
		if err != nil {
			return err
		}

		//fmt.Printf("seeds:\n%v\nmaskedT:\n%v\nmaskedU\n%v\n", baseMsgs[row], maskedTj, maskedUj)
		// send t^j
		if _, err = rw.Write(maskedTj); err != nil {
			return err
		}

		// send u^j
		if _, err = rw.Write(maskedUj); err != nil {
			return err
		}
	}

	//fmt.Printf("t:\n%v\nu:\n%v\n", t, u)
	t = transpose(t)

	// receive encrypted messages.
	e := make([][]byte, 2)
	for i := range choices {
		// compute # of bytes to be read
		l := encryptLen(iknpCipherMode, ext.msgLen[i])
		// read both msg
		for j, _ := range e {
			e[j] = make([]byte, l)
			if _, err = io.ReadFull(rw, e[j]); err != nil {
				return err
			}
		}

		// decrypt received ciphertext using key (choices[i], t_i)
		messages[i], err = decrypt(iknpCipherMode, t[i], choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("Error decrypting sender messages: %s\n", err)
		}
	}

	return
}
