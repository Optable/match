package util

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

var benchmarkBytes = 10000000

func genBytes(size int) []byte {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		panic("error generating random bytes")
	}

	return bytes
}

type bitOpResult struct {
	A      []byte
	B      []byte
	C      []byte
	Xor    []byte // A XOR B
	And    []byte // A AND B
	XorXor []byte // A XOR B XOR C
	AndXor []byte // A AND B XOR C
}

func (bitOpResult) Generate(r *rand.Rand, size int) reflect.Value {
	var result bitOpResult
	// generate initial random slices
	result.A = genBytes(size)
	result.B = genBytes(size)
	result.C = genBytes(size)
	// XOR
	result.Xor = make([]byte, size)
	for i := range result.Xor {
		result.Xor[i] = result.A[i] ^ result.B[i]
	}

	// AND
	result.And = make([]byte, size)
	for i := range result.And {
		result.And[i] = result.A[i] & result.B[i]
	}

	// Double XOR
	result.XorXor = make([]byte, size)
	for i := range result.XorXor {
		result.XorXor[i] = result.Xor[i] ^ result.C[i]
	}

	// AND XOR
	result.AndXor = make([]byte, size)
	for i := range result.AndXor {
		result.AndXor[i] = result.And[i] ^ result.C[i]
	}

	return reflect.ValueOf(result)
}

func TestXor(t *testing.T) {
	correct := func(b bitOpResult) bool {
		err := Xor(b.A, b.B)
		for i := range b.Xor {
			if b.A[i] != b.Xor[i] {
				return false
			}
		}
		return err == nil
	}

	if err := quick.Check(correct, nil); err != nil {
		t.Errorf("bitwise XOR fail: %v", err)
	}
}

func TestAnd(t *testing.T) {
	correct := func(b bitOpResult) bool {
		err := And(b.A, b.B)
		for i := range b.And {
			if b.A[i] != b.And[i] {
				return false
			}
		}
		return err == nil
	}

	if err := quick.Check(correct, nil); err != nil {
		t.Errorf("bitwise AND fail: %v", err)
	}
}

func TestDoubleXor(t *testing.T) {
	correct := func(b bitOpResult) bool {
		err := DoubleXor(b.A, b.B, b.C)
		for i := range b.XorXor {
			if b.A[i] != b.XorXor[i] {
				return false
			}
		}
		return err == nil
	}

	if err := quick.Check(correct, nil); err != nil {
		t.Errorf("bitwise double XOR fail: %v", err)
	}
}

func TestAndXor(t *testing.T) {
	correct := func(b bitOpResult) bool {
		err := AndXor(b.A, b.B, b.C)
		for i := range b.AndXor {
			if b.A[i] != b.AndXor[i] {
				return false
			}
		}
		return err == nil
	}

	if err := quick.Check(correct, nil); err != nil {
		t.Errorf("bitwise AND followed by XOR fail: %v", err)
	}
}

func TestConcurrentBitOp(t *testing.T) {
	xor := func(b bitOpResult) bool {
		err := ConcurrentBitOp(Xor, b.A, b.B)
		for i := range b.Xor {
			if b.A[i] != b.Xor[i] {
				return false
			}
		}
		return err == nil
	}

	if err := quick.Check(xor, nil); err != nil {
		t.Errorf("concurrent bitwise XOR fail: %v", err)
	}

	and := func(b bitOpResult) bool {
		err := ConcurrentBitOp(And, b.A, b.B)
		for i := range b.And {
			if b.A[i] != b.And[i] {
				return false
			}
		}
		return err == nil
	}

	if err := quick.Check(and, nil); err != nil {
		t.Errorf("concurrent bitwise AND fail: %v", err)
	}
}

func TestConcurrentDoubleBitOp(t *testing.T) {
	xorxor := func(b bitOpResult) bool {
		err := ConcurrentDoubleBitOp(DoubleXor, b.A, b.B, b.C)
		for i := range b.XorXor {
			if b.A[i] != b.XorXor[i] {
				return false
			}
		}
		return err == nil
	}

	if err := quick.Check(xorxor, nil); err != nil {
		t.Errorf("concurrent bitwise double XOR fail: %v", err)
	}

	andxor := func(b bitOpResult) bool {
		err := ConcurrentDoubleBitOp(AndXor, b.A, b.B, b.C)
		for i := range b.AndXor {
			if b.A[i] != b.AndXor[i] {
				return false
			}
		}
		return err == nil
	}

	if err := quick.Check(andxor, nil); err != nil {
		t.Errorf("concurrent bitwise AND followed by XOR fail: %v", err)
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
