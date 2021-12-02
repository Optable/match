package util

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

var benchmarkBytes = 10

func genBytes(size int) []byte {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		panic("error generating random bytes")
	}

	return bytes
}

type bitSets struct {
	Scratch []byte
	A       []byte
	B       []byte
	C       []byte
}

// Generate creates a bitSets struct with three byte slices of
// equal length
func (bitSets) Generate(r *rand.Rand, size int) reflect.Value {
	var sets bitSets
	sets.Scratch = make([]byte, size)
	sets.A = genBytes(size)
	sets.B = genBytes(size)
	sets.C = genBytes(size)
	return reflect.ValueOf(sets)
}

func TestXor(t *testing.T) {
	fast := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		Xor(b.Scratch, b.B)
		return b.Scratch
	}

	naive := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		for i := range b.Scratch {
			b.Scratch[i] ^= b.B[i]
		}
		return b.Scratch
	}

	if err := quick.CheckEqual(fast, naive, nil); err != nil {
		t.Errorf("fast XOR != naive XOR: %v", err)
	}

	commutative := func(b bitSets) bool {
		copy(b.Scratch, b.A)
		Xor(b.Scratch, b.B)
		Xor(b.B, b.A)
		// check
		for i := range b.Scratch {
			if b.Scratch[i] != b.B[i] {
				return false
			}
		}
		return true
	}

	if err := quick.Check(commutative, nil); err != nil {
		t.Errorf("A ^ B != B ^ A (commutative): %v", err)
	}

	associative := func(b bitSets) bool {
		copy(b.Scratch, b.B)
		Xor(b.Scratch, b.C)
		Xor(b.Scratch, b.A)

		// check
		Xor(b.A, b.B)
		Xor(b.A, b.C)

		for i := range b.Scratch {
			if b.Scratch[i] != b.A[i] {
				return false
			}
		}
		return true
	}

	if err := quick.Check(associative, nil); err != nil {
		t.Errorf("A ^ (B ^ C) != (A ^ B) ^ C (associative): %v", err)
	}

	identityElement := func(b bitSets) bool {
		Xor(b.Scratch, b.A)
		// check
		for i := range b.Scratch {
			if b.Scratch[i] != b.A[i] {
				return false
			}
		}
		return true
	}

	if err := quick.Check(identityElement, nil); err != nil {
		t.Errorf("A ^ 0 != A (identity): %v", err)
	}

	selfInverse := func(b bitSets) bool {
		Xor(b.A, b.A)
		// check
		for i := range b.A {
			if b.A[i] != 0 {
				return false
			}
		}
		return true
	}

	if err := quick.Check(selfInverse, nil); err != nil {
		t.Errorf("A ^ A != 0 (self-inverse): %v", err)
	}
}

func TestAnd(t *testing.T) {
	fast := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		And(b.Scratch, b.B)
		return b.Scratch
	}

	naive := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		for i := range b.Scratch {
			b.Scratch[i] &= b.B[i]
		}
		return b.Scratch
	}

	if err := quick.CheckEqual(fast, naive, nil); err != nil {
		t.Errorf("fast AND != naive AND: %v", err)
	}

	annulment := func(b bitSets) bool {
		And(b.Scratch, b.A)
		// check
		for i := range b.Scratch {
			if b.Scratch[i] != 0 {
				return false
			}
		}
		return true
	}

	if err := quick.Check(annulment, nil); err != nil {
		t.Errorf("A & 0 != 0 (annulment): %v", err)
	}

	commutative := func(b bitSets) bool {
		copy(b.Scratch, b.A)
		And(b.Scratch, b.B)
		And(b.B, b.A)
		// check
		for i := range b.Scratch {
			if b.Scratch[i] != b.B[i] {
				return false
			}
		}
		return true
	}

	if err := quick.Check(commutative, nil); err != nil {
		t.Errorf("A & B != B & A (commutative): %v", err)
	}

	associative := func(b bitSets) bool {
		copy(b.Scratch, b.B)
		And(b.Scratch, b.C)
		And(b.Scratch, b.A)

		// check
		And(b.A, b.B)
		And(b.A, b.C)

		for i := range b.Scratch {
			if b.Scratch[i] != b.A[i] {
				return false
			}
		}
		return true
	}

	if err := quick.Check(associative, nil); err != nil {
		t.Errorf("A & (B & C) != (A & B) & C (associative): %v", err)
	}

	identityElement := func(b bitSets) bool {
		for i := range b.Scratch {
			b.Scratch[i] = 255
		}
		And(b.Scratch, b.A)
		// check
		for i := range b.Scratch {
			if b.Scratch[i] != b.A[i] {
				return false
			}
		}
		return true
	}

	if err := quick.Check(identityElement, nil); err != nil {
		t.Errorf("A & 1 != A (identity): %v", err)
	}

	idempotent := func(b bitSets) bool {
		copy(b.Scratch, b.A)
		And(b.Scratch, b.A)
		// check
		for i := range b.Scratch {
			if b.Scratch[i] != b.A[i] {
				return false
			}
		}
		return true
	}

	if err := quick.Check(idempotent, nil); err != nil {
		t.Errorf("A & A != A (idempotent): %v", err)
	}
}

func TestDoubleXor(t *testing.T) {
	fast := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		DoubleXor(b.Scratch, b.B, b.C)
		return b.Scratch
	}

	naive := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		for i := range b.Scratch {
			b.Scratch[i] &= b.B[i]
			b.Scratch[i] &= b.C[i]
		}
		return b.Scratch
	}

	if err := quick.CheckEqual(fast, naive, nil); err != nil {
		t.Errorf("fast double XOR != naive double XOR: %v", err)
	}
}

func TestAndXor(t *testing.T) {
	fast := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		AndXor(b.Scratch, b.B, b.C)
		return b.Scratch
	}

	naive := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		for i := range b.Scratch {
			b.Scratch[i] &= b.B[i]
			b.Scratch[i] ^= b.C[i]
		}
		return b.Scratch
	}

	if err := quick.CheckEqual(fast, naive, nil); err != nil {
		t.Errorf("fast AND followed by XOR != naive AND followed by XOR: %v", err)
	}
}

func TestConcurrentBitOp(t *testing.T) {
	concXor := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		ConcurrentBitOp(Xor, b.Scratch, b.B)
		return b.Scratch
	}

	naiveXor := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		for i := range b.Scratch {
			b.Scratch[i] ^= b.B[i]
		}
		return b.Scratch
	}

	if err := quick.CheckEqual(concXor, naiveXor, nil); err != nil {
		t.Errorf("concurrent fast XOR != naive XOR: %v", err)
	}

	concAnd := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		ConcurrentBitOp(And, b.Scratch, b.B)
		return b.Scratch
	}

	naiveAnd := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		for i := range b.Scratch {
			b.Scratch[i] &= b.B[i]
		}
		return b.Scratch
	}

	if err := quick.CheckEqual(concAnd, naiveAnd, nil); err != nil {
		t.Errorf("concurrent fast AND != naive AND: %v", err)
	}
}

func TestConcurrentDoubleBitOp(t *testing.T) {
	concDoubleXor := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		ConcurrentDoubleBitOp(DoubleXor, b.Scratch, b.B, b.C)
		return b.Scratch
	}

	naiveDoubleXor := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		for i := range b.Scratch {
			b.Scratch[i] ^= b.B[i]
			b.Scratch[i] ^= b.C[i]
		}
		return b.Scratch
	}

	if err := quick.CheckEqual(concDoubleXor, naiveDoubleXor, nil); err != nil {
		t.Errorf("concurrent fast double XOR != naive double XOR: %v", err)
	}

	concAndXor := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		ConcurrentDoubleBitOp(AndXor, b.Scratch, b.B, b.C)
		return b.Scratch
	}

	naiveAndXor := func(b bitSets) []byte {
		copy(b.Scratch, b.A)
		for i := range b.Scratch {
			b.Scratch[i] &= b.B[i]
			b.Scratch[i] ^= b.C[i]
		}
		return b.Scratch
	}

	if err := quick.CheckEqual(concAndXor, naiveAndXor, nil); err != nil {
		t.Errorf("concurrent fast AND followed by XOR != naive AND followed by XOR: %v", err)
	}
}

func TestTestBitSetInByte(t *testing.T) {
	b := []byte{1}

	for i := 0; i < 8; i++ {
		if i == 0 {
			if !IsBitSet(b, i) {
				t.Fatal("bit extraction failed")
			}
		} else {
			if IsBitSet(b, i) {
				t.Fatal("bit extraction failed")
			}
		}
	}

	b = []byte{161}
	for i := 0; i < 8; i++ {
		if i == 0 || i == 7 || i == 5 {
			if !IsBitSet(b, i) {
				t.Fatal("bit extraction failed")
			}
		} else {
			if IsBitSet(b, i) {
				t.Fatal("bit extraction failed")
			}
		}
	}
}

func BenchmarkXor(b *testing.B) {
	src := genBytes(benchmarkBytes)
	dst := genBytes(benchmarkBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Xor(dst, src)
	}
}

func BenchmarkAnd(b *testing.B) {
	src := genBytes(benchmarkBytes)
	dst := genBytes(benchmarkBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		And(dst, src)
	}
}

func BenchmarkDoubleXor(b *testing.B) {
	src := genBytes(benchmarkBytes)
	src2 := genBytes(benchmarkBytes)
	dst := genBytes(benchmarkBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DoubleXor(dst, src, src2)
	}
}

func BenchmarkAndXor(b *testing.B) {
	src := genBytes(benchmarkBytes)
	src2 := genBytes(benchmarkBytes)
	dst := genBytes(benchmarkBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AndXor(dst, src, src2)
	}
}

func BenchmarkConcurrentBitOp(b *testing.B) {
	src := genBytes(benchmarkBytes)
	dst := genBytes(benchmarkBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentBitOp(Xor, dst, src)
	}
}

func BenchmarkConcurrentDoubleBitOp(b *testing.B) {
	src := genBytes(benchmarkBytes)
	src2 := genBytes(benchmarkBytes)
	dst := genBytes(benchmarkBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentDoubleBitOp(AndXor, dst, src, src2)
	}
}
