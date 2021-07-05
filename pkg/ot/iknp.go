package ot

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"time"
)

const (
	iknpCurve      = "P256"
	iknpCipherMode = XORBlake3
)

type iknp struct {
	baseOt Ot
	m      int
	k      int
	msgLen []int
	prng   *rand.Rand
}

func NewIknp(m, k, baseOt int, ristretto bool, msgLen []int) (iknp, error) {
	// send k columns of messages of length m
	baseMsgLen := make([]int, k)
	for i, _ := range baseMsgLen {
		baseMsgLen[i] = m
	}

	ot, err := NewBaseOt(baseOt, ristretto, k, iknpCurve, baseMsgLen, iknpCipherMode)
	if err != nil {
		return iknp{}, err
	}

	return iknp{baseOt: ot, m: m, k: k, msgLen: msgLen, prng: rand.New(rand.NewSource(time.Now().UnixNano()))}, nil
}

func (ext iknp) Send(messages [][2][]byte, rw io.ReadWriter) (err error) {
	// sample choice bits for baseOT
	s := make([]uint8, ext.k)
	if err = sampleBitSlice(ext.prng, s); err != nil {
		return err
	}

	// act as receiver in baseOT to receive q^j
	q := make([][]uint8, ext.k)
	if err = ext.baseOt.Receive(s, q, rw); err != nil {
		return err
	}

	// transpose q to m x k matrix for easier row operations
	q = transpose(q)

	var key, ciphertext []byte
	// encrypt messages and send them
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

func (ext iknp) Receive(choices []uint8, messages [][]byte, rw io.ReadWriter) (err error) {
	if len(choices) != len(messages) || len(choices) != ext.m {
		return ErrBaseCountMissMatch
	}

	// Sample k x m matrix T
	t, err := sampleRandomBitMatrix(ext.prng, ext.k, ext.m)
	if err != nil {
		return err
	}

	// make k pairs of m bytes baseOT messages: {t^j, t^j xor choices}
	baseMsgs := make([][2][]byte, ext.k)
	for j := range baseMsgs {
		// []uint8 = []byte, since byte is an alias to uint8
		baseMsgs[j][0] = t[j]
		baseMsgs[j][1], err = xorBytes(t[j], choices)
		if err != nil {
			return err
		}
	}

	// act as sender in baseOT to send k columns
	if err = ext.baseOt.Send(baseMsgs, rw); err != nil {
		return err
	}

	// compute k x m transpose to access columns easier
	t = transpose(t)

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

func computeMaskedRows(r *rand.Rand, s1, s2, p1, p2 []byte) (c1, c2 []byte, err error) {
	k := len(s1)
	mask1 := make([]byte, k)
	mask2 := make([]byte, k)
	if err = sampleBitSliceWithSeed(r, s1, mask1); err != nil {
		return nil, nil, err
	}

	if err = sampleBitSliceWithSeed(r, s2, mask2); err != nil {
		return nil, nil, err
	}

	c1, err = xorBytes(p1, mask1)
	if err != nil {
		return nil, nil, err
	}

	c2, err = xorBytes(p2, mask2)
	if err != nil {
		return nil, nil, err
	}

	return
}

// transpose returns the transpose of a 2D slices of uint8
// from (m x k) to (k x m)
func transpose(matrix [][]uint8) [][]uint8 {
	n := len(matrix)
	tr := make([][]uint8, len(matrix[0]))

	for row := range tr {
		tr[row] = make([]uint8, n)
		for col := range tr[row] {
			tr[row][col] = matrix[col][row]
		}
	}
	return tr
}

// sampleRandomBitMatrix fills each entry in the given 2D slices of uint8
// with pseudorandom bit values
func sampleRandomBitMatrix(r *rand.Rand, m, k int) ([][]uint8, error) {
	// instantiate matrix
	matrix := make([][]uint8, m)
	for row := range matrix {
		matrix[row] = make([]uint8, k)
	}

	for row := range matrix {
		if err := sampleBitSlice(r, matrix[row]); err != nil {
			return nil, err
		}
	}

	return matrix, nil
}

// sampleBitSliceWithSeed returns a slice of uint8 of seeded pseudorandom bits
func sampleBitSliceWithSeed(r *rand.Rand, seed []byte, b []uint8) (err error) {
	r.Seed(int64(binary.BigEndian.Uint64(seed)))
	return sampleBitSlice(r, b)
}

// sampleBitSlice returns a slice of uint8 of pseudorandom bits
func sampleBitSlice(prng *rand.Rand, b []uint8) (err error) {
	// read up to len(b) pseudorandom bits
	t := make([]byte, len(b)/8)
	if _, err = prng.Read(t); err != nil {
		return nil
	}

	// extract all bits into b
	extractBytesToBits(t, b)

	return nil
}

func extractBytesToBits(t, b []byte) {
	var i int
	for _, _byte := range t {
		b[i] = uint8(_byte & 0x01)
		b[i+1] = uint8((_byte >> 1) & 0x01)
		b[i+2] = uint8((_byte >> 2) & 0x01)
		b[i+3] = uint8((_byte >> 3) & 0x01)
		b[i+4] = uint8((_byte >> 4) & 0x01)
		b[i+5] = uint8((_byte >> 5) & 0x01)
		b[i+6] = uint8((_byte >> 6) & 0x01)
		b[i+7] = uint8((_byte >> 7) & 0x01)
		i += 8
	}
}
