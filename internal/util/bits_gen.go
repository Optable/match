// +build !amd64 generic

package util

import (
	"encoding/binary"
)

// Xor casts the first part of the byte slice (length divisible
// by 8) into uint64 and then performs XOR on the slice of uint64.
// The excess elements that could not be cast are XORed conventionally.
// The whole operation is performed in place. Panic if a and dst do
// not have the same length
// Of course a and dst must be the same length and the whole operation
// is performed in place.
func Xor(dst, a []byte) {
	if len(dst) != len(a) {
		panic(ErrByteLengthMissMatch)
	}

	// process as uint64 when possible
	var uDst, uA uint64
	for i := 0; i < len(dst)/8; i++ {
		uDst = binary.LittleEndian.Uint64(dst[i*8 : (i+1)*8])
		uA = binary.LittleEndian.Uint64(a[i*8 : (i+1)*8])
		binary.LittleEndian.PutUint64(dst[i*8:(i+1)*8], uDst^uA)
	}

	// deal with excess bytes that couldn't be operated
	// as uint64s
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] ^= a[len(dst)-j-1]
	}
}

// And casts the first part of the byte slice (length divisible
// by 8) into uint64 and then performs AND on the slice of uint64.
// The excess elements that could not be cast are ANDed conventionally.
// The whole operation is performed in place. Panic if a and dst do
// not have the same length.
func And(dst, a []byte) {
	if len(dst) != len(a) {
		panic(ErrByteLengthMissMatch)
	}

	// process as uint64 when possible
	var uDst, uA uint64
	for i := 0; i < len(dst)/8; i++ {
		uDst = binary.LittleEndian.Uint64(dst[i*8 : (i+1)*8])
		uA = binary.LittleEndian.Uint64(a[i*8 : (i+1)*8])
		binary.LittleEndian.PutUint64(dst[i*8:(i+1)*8], uDst&uA)
	}

	// deal with excess bytes that couldn't be operated
	// as uint64s
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] &= a[len(dst)-j-1]
	}
}

// DoubleXor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs XOR on the slices of uint64
// (first with a and then with b). The excess elements that could not
// be cast are XORed conventionally. The whole operation is performed
// in place. Panic if a, b and dst do not have the same length.
func DoubleXor(dst, a, b []byte) {
	if len(dst) != len(a) || len(dst) != len(b) {
		panic(ErrByteLengthMissMatch)
	}

	// process as uint64 when possible
	var uDst, uA, uB uint64
	for i := 0; i < len(dst)/8; i++ {
		uDst = binary.LittleEndian.Uint64(dst[i*8 : (i+1)*8])
		uA = binary.LittleEndian.Uint64(a[i*8 : (i+1)*8])
		uB = binary.LittleEndian.Uint64(b[i*8 : (i+1)*8])
		binary.LittleEndian.PutUint64(dst[i*8:(i+1)*8], uDst^uA^uB)
	}

	// deal with excess bytes that couldn't be operated
	// as uint64s
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] ^= a[len(dst)-j-1]
		dst[len(dst)-j-1] ^= b[len(dst)-j-1]
	}
}

// AndXor casts the first part of the byte slices (length divisible
// by 8) into uint64 and then performs AND on the slices of uint64
// (with a) and then performs XOR (with b). The excess elements
// That could not be cast are oeprated on conventionally. The whole
// operation is performed in place. Panic if a, b and dst do not
// have the same length.
func AndXor(dst, a, b []byte) {
	if len(dst) != len(a) || len(dst) != len(b) {
		panic(ErrByteLengthMissMatch)
	}

	// process as uint64 when possible
	var uDst, uA, uB uint64
	for i := 0; i < len(dst)/8; i++ {
		uDst = binary.LittleEndian.Uint64(dst[i*8 : (i+1)*8])
		uA = binary.LittleEndian.Uint64(a[i*8 : (i+1)*8])
		uB = binary.LittleEndian.Uint64(b[i*8 : (i+1)*8])
		binary.LittleEndian.PutUint64(dst[i*8:(i+1)*8], uDst&uA^uB)
	}

	// deal with excess bytes that couldn't be operated
	// as uint64s
	for j := 0; j < len(dst)%8; j++ {
		dst[len(dst)-j-1] &= a[len(dst)-j-1]
		dst[len(dst)-j-1] ^= b[len(dst)-j-1]
	}
}
