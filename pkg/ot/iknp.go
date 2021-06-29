package ot

import (
	"fmt"
	"io"
	"math/rand"
	"time"
)

const (
	iknpCurve      = "p256"
	iknpCipherMode = XOR
)

type iknp struct {
	baseOt Ot
	m      int
	k      int
	msgLen []int
	prng   *rand.Rand
}

func newIknp(m, k, baseOt int, ristretto bool, msgLen []int) (iknp, error) {
	// m x k matrix, but send and receive the columns.
	baseMsgLen := make([]int, k)
	for i, _ := range baseMsgLen {
		baseMsgLen[i] = m
	}

	ot, err := NewBaseOt(baseOt, ristretto, k, iknpCurve, baseMsgLen, iknpCipherMode)
	if err != nil {
		return iknp{}, err
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return iknp{baseOt: ot, m: m, k: k, msgLen: msgLen, prng: r}, nil
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

	// Sample m x k matrix T
	T := make([][]uint8, ext.m)
	for row := range T {
		T[row] = make([]uint8, ext.k)
	}

	if err = sampleRandomBitMatrix(ext.prng, T); err != nil {
		return err
	}

	// compute k x m transpose to access columns easier
	Tt := transpose(T)

	// make k pairs of m bytes baseOT messages: {T^j, T^j xor choices}
	baseMsgs := make([][2][]byte, ext.k)
	for j := range baseMsgs {
		// []uint8 = []byte, since byte is an alias to uint8
		baseMsgs[j][0] = Tt[j]
		// method from cipher.go
		baseMsgs[j][1], err = xorBytes(Tt[j], choices)
		if err != nil {
			return err
		}
	}

	// ready to do baseOT, act as sender to send the columns
	if err = ext.baseOt.Send(baseMsgs, rw); err != nil {
		return err
	}

	e := make([][]byte, 2)
	// receive encrypted messages.
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
		messages[i], err = decrypt(iknpCipherMode, T[i], choices[i], e[choices[i]])
		if err != nil {
			return fmt.Errorf("Error decrypting sender messages: %s\n", err)
		}
	}

	return
}

// transpose returns the transpose of a 2D slices of *big.Int
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

// sampleRandomBitMatrix takes a 2D slices of *big.Int
// and calls crypto/rand.Int(2) for each slot in the matrix.
// slightly expensive operation, maybe math/rand suffices
// We might benefit from fitting bits in byte slices, and extracting them later on?
func sampleRandomBitMatrix(prng *rand.Rand, matrix [][]uint8) (err error) {
	for row := range matrix {
		if err = sampleBitSlice(prng, matrix[row]); err != nil {
			return err
		}
	}

	return
}

// sampleBitSlice returns a slice of uint8 of pseudorandom bits.
func sampleBitSlice(prng *rand.Rand, b []uint8) (err error) {
	if _, err = prng.Read(b); err != nil {
		return err
	}
	for i := range b {
		b[i] = uint8(b[i]) % 2
	}

	return
}
