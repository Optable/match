// +build amd64,!generic

package util

import (
	"github.com/alecthomas/unsafeslice"
)

// Xor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs XOR on the slices of uint64.
// The excess elements that could not be cast are XORed conventionally.
// The whole operation is performed in place. Panic if a and dst do
// not have the same length.
// Only tested on x86-64.
func Xor(dst, a []byte) {
	if len(dst) != len(a) {
		panic(ErrByteLengthMissMatch)
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)

	for i := range castDst {
		castDst[i] ^= castA[i]
	}

	// deal with excess bytes which could not be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] ^= a[len(a)-j-1]
	}
}

// And casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs AND on the slices of uint64.
// The excess elements that could not be cast are ANDed conventionally.
// The whole operation is performed in place. Panic if a and dst do
// not have the same length.
// Only tested on x86-64.
func And(dst, a []byte) {
	if len(dst) != len(a) {
		panic(ErrByteLengthMissMatch)
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)

	for i := range castDst {
		castDst[i] &= castA[i]
	}

	// deal with excess bytes which could not be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] &= a[len(a)-j-1]
	}
}

// DoubleXor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs XOR on the slices of uint64
// (first with a and then with b). The excess elements that could not
// be cast are XORed conventionally. The whole operation is performed
// in place. Panic if a, b and dst do not have the same length.
// Only tested on x86-64.
func DoubleXor(dst, a, b []byte) {
	if len(dst) != len(a) || len(dst) != len(b) {
		panic(ErrByteLengthMissMatch)
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)
	castB := unsafeslice.Uint64SliceFromByteSlice(b)

	for i := range castDst {
		castDst[i] ^= castA[i]
		castDst[i] ^= castB[i]
	}

	// deal with excess bytes which could not be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] ^= a[len(a)-j-1]
		dst[len(dst)-j-1] ^= b[len(b)-j-1]
	}
}

// AndXor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs AND on the slices of uint64
// (with a) and then performs XOR (with b). The excess elements
// that could not be cast are operated on conventionally. The whole
// operation is performed in place. Panic if a, b and dst do not
// have the same length.
// Only tested on x86-64.
func AndXor(dst, a, b []byte) {
	if len(dst) != len(a) || len(dst) != len(b) {
		panic(ErrByteLengthMissMatch)
	}

	castDst := unsafeslice.Uint64SliceFromByteSlice(dst)
	castA := unsafeslice.Uint64SliceFromByteSlice(a)
	castB := unsafeslice.Uint64SliceFromByteSlice(b)

	for i := range castDst {
		castDst[i] &= castA[i]
		castDst[i] ^= castB[i]
	}

	// deal with excess bytes which could not be cast to uint64
	// in the conventional manner
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] &= a[len(a)-j-1]
		dst[len(dst)-j-1] ^= b[len(b)-j-1]
	}
}
