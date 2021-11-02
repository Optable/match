package util

import (
	"bytes"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/alecthomas/unsafeslice"
)

var prng = rand.New(rand.NewSource(time.Now().UnixNano()))

func sampleByteSlice(prng *rand.Rand, b []byte) (err error) {
	if _, err = prng.Read(b); err != nil {
		return nil
	}
	return nil
}

func sampleUint64Slice(prng *rand.Rand, u []uint64) {
	for i := range u {
		u[i] = prng.Uint64()
	}
}

// naiveXorBytes XORS each byte from a with b and returns dst
// if a and b are the same length
func naiveXorBytes(a, b []byte) (dst []byte, err error) {
	var n = len(b)
	if n != len(a) {
		return nil, ErrByteLengthMissMatch
	}

	dst = make([]byte, n)

	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}

	return
}

// inPlaceXorBytes XORS each byte from a with dst in place
// if a and dst are the same length
func inPlaceXorBytes(dst, a []byte) error {
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := 0; i < n; i++ {
		dst[i] ^= a[i]
	}

	return nil
}

// concurrentInPlaceXorBytes XORS each byte from a with dst in place
// if a and dst are the same length
func concurrentInPlaceXorBytes(dst, a []byte) error {
	const blockSize int = 16384 // half of what L2 cache can hold
	nworkers := runtime.GOMAXPROCS(0)
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
	if n < blockSize {
		return inPlaceXorBytes(dst, a)
	}

	// determine number of blocks to split original matrix
	nblks := n / blockSize
	if n%blockSize != 0 {
		nblks += 1
	}
	ch := make(chan int, nblks)
	for i := 0; i < nblks; i++ {
		ch <- i
	}
	close(ch)

	// Run a worker pool
	var wg sync.WaitGroup
	wg.Add(nworkers)
	for w := 0; w < nworkers; w++ {
		go func() {
			defer wg.Done()
			for blk := range ch {
				step := blockSize * blk
				if blk == nblks-1 { // last block
					for i := step; i < n; i++ {
						dst[i] ^= a[i]
					}
				} else {
					for i := step; i < step+blockSize; i++ {
						dst[i] ^= a[i]
					}
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

// inPlaceAndBytes performs the binary AND of each byte in a
// and dst in place if a and dst are the same length.
func inPlaceAndBytes(dst, a []byte) error {
	n := len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	for i := range dst {
		dst[i] = dst[i] & a[i]
	}

	return nil
}

// concurrentInPlaceAndBytes performs the binary AND of each
// byte in a and dst if a and dst are the same length.
func concurrentInPlaceAndBytes(dst, a []byte) error {
	const blockSize int = 16384 // half of what L2 cache can hold
	nworkers := runtime.GOMAXPROCS(0)
	var n = len(dst)
	if n != len(a) {
		return ErrByteLengthMissMatch
	}

	// no need to split into goroutines
	if n < blockSize {
		return inPlaceAndBytes(dst, a)
	}

	// determine number of blocks to split original matrix
	nblks := n / blockSize
	if n%blockSize != 0 {
		nblks += 1
	}
	ch := make(chan int, nblks)
	for i := 0; i < nblks; i++ {
		ch <- i
	}
	close(ch)

	// Run a worker pool
	var wg sync.WaitGroup
	wg.Add(nworkers)
	for w := 0; w < nworkers; w++ {
		go func() {
			defer wg.Done()
			for blk := range ch {
				step := blockSize * blk
				if blk == nblks-1 { // last block
					for i := step; i < n; i++ {
						dst[i] &= a[i]
					}
				} else {
					for i := step; i < step+blockSize; i++ {
						dst[i] &= a[i]
					}
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

// naiveTranspose returns the transpose of a 2D slices of bytes
// from (m x k) to (k x m) by naively swapping.
func naiveTranspose(matrix [][]uint8) [][]uint8 {
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

func TestTestBitSetInByte(t *testing.T) {
	b := []byte{1}

	for i := 0; i < 8; i++ {
		if i == 0 {
			if !BitSetInByte(b, i) {
				t.Fatalf("bit extraction failed")
			}
		} else {
			if BitSetInByte(b, i) {
				t.Fatalf("bit extraction failed")
			}
		}
	}

	b = []byte{161}
	for i := 0; i < 8; i++ {
		if i == 0 || i == 7 || i == 5 {
			if !BitSetInByte(b, i) {
				t.Fatalf("bit extraction failed")
			}
		} else {
			if BitSetInByte(b, i) {
				t.Fatalf("bit extraction failed")
			}
		}

	}
}

// Note the double conversion of bytes to uint64s to bytes does
// result in added 0s.
func TestSliceConversions(t *testing.T) {
	lengths := []int{8, 16, 24, 32, 40, 48}
	for _, l := range lengths {
		// Bytes to Uint64s
		b := make([]byte, l)
		sampleByteSlice(prng, b)
		u := unsafeslice.Uint64SliceFromByteSlice(b)
		bb := unsafeslice.ByteSliceFromUint64Slice(u)

		// test
		for i, e := range b {
			if e != bb[i] {
				t.Errorf("Byte-to-Uint64-to-Byte conversion did not result in identical slices")
			}
		}
	}
	lengths = []int{2, 8, 16, 34, 100}
	for _, l := range lengths {
		// Uint64s to Bytes
		u := make([]uint64, l)
		sampleUint64Slice(prng, u)
		b := unsafeslice.ByteSliceFromUint64Slice(u)
		uu := unsafeslice.Uint64SliceFromByteSlice(b)

		//test
		for i, e := range u {
			if e != uu[i] {
				t.Errorf("Uint64-to-Byte-to-Uint64 conversion did not result in identical slices")
			}
		}

	}
}

func TestNaiveTranspose(t *testing.T) {
	b := make([][]byte, 4)
	for m := range b {
		b[m] = make([]byte, 8)
		prng.Read(b[m])
	}

	for m := range b {
		if !bytes.Equal(b[m], naiveTranspose(naiveTranspose(b))[m]) {
			t.Fatalf("Transpose of transpose should be equal")
		}
	}
}

func TestConcurrentInPlaceXorBytes(t *testing.T) {
	for _, l := range []int{10, 16384, 10000000} {
		// XOR with itself
		a := make([]byte, l)
		if _, err := prng.Read(a); err != nil {
			t.Fatalf("error generating random bytes")
		}
		concurrentInPlaceXorBytes(a, a)
		for _, i := range a {
			if i != 0 {
				t.Fatalf("XOR operation was not performed correctly")
			}
		}
		// doubly XOR with another slice to get back original
		c := make([]byte, l)
		d := make([]byte, l)
		e := make([]byte, l)
		if _, err := prng.Read(c); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(e); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(d, c) // save original to check later
		concurrentInPlaceXorBytes(c, e)
		concurrentInPlaceXorBytes(c, e)
		for i := range c {
			if c[i] != d[i] {
				t.Fatalf("performing concurrent XOR operation twice did not result in same slice")
			}
		}
		// XOR same original slice with concurrent and non-concurrent versions and then compare output
		f := make([]byte, l)
		g := make([]byte, l)
		h := make([]byte, l)
		if _, err := prng.Read(f); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(h); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(g, f)
		concurrentInPlaceXorBytes(f, h)
		inPlaceXorBytes(g, h)
		for i := range f {
			if f[i] != g[i] {
				t.Fatalf("result of concurrent XOR operation did not match with result of non-concurrent equivalent")
			}
		}
	}
}

func TestConcurrentUnsafeInPlaceXorBytes(t *testing.T) {
	for _, l := range []int{10, 16384, 10000000} {
		// XOR with itself
		a := make([]byte, l)
		if _, err := prng.Read(a); err != nil {
			t.Fatalf("error generating random bytes")
		}
		ConcurrentBitOp(Xor, a, a)
		for _, i := range a {
			if i != 0 {
				t.Fatalf("XOR operation was not performed correctly")
			}
		}
		// doubly XOR with another slice to get back original
		c := make([]byte, l)
		d := make([]byte, l)
		e := make([]byte, l)
		if _, err := prng.Read(c); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(e); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(d, c) // save original to check later
		ConcurrentBitOp(Xor, c, e)
		ConcurrentBitOp(Xor, c, e)
		for i := range c {
			if c[i] != d[i] {
				t.Fatalf("performing concurrent XOR operation twice did not result in same slice")
			}
		}
		// XOR same original slice with concurrent and non-concurrent versions and then compare output
		f := make([]byte, l)
		g := make([]byte, l)
		h := make([]byte, l)
		if _, err := prng.Read(f); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(h); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(g, f)
		ConcurrentBitOp(Xor, f, h)
		inPlaceXorBytes(g, h)
		for i := range f {
			if f[i] != g[i] {
				t.Fatalf("result of concurrent XOR operation did not match with result of non-concurrent equivalent")
			}
		}
	}
}

func TestUnsafeInPlaceXorBytes(t *testing.T) {
	for _, l := range []int{10, 16384, 10000000} {
		// XOR with itself
		a := make([]byte, l)
		if _, err := prng.Read(a); err != nil {
			t.Fatalf("error generating random bytes")
		}
		Xor(a, a)
		for _, i := range a {
			if i != 0 {
				t.Fatalf("XOR operation was not performed correctly")
			}
		}
		// doubly XOR with another slice to get back original
		c := make([]byte, l)
		d := make([]byte, l)
		e := make([]byte, l)
		if _, err := prng.Read(c); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(e); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(d, c) // save original to check later
		Xor(c, e)
		Xor(c, e)
		for i := range c {
			if c[i] != d[i] {
				t.Fatalf("performing concurrent XOR operation twice did not result in same slice")
			}
		}
		// XOR same original slice with concurrent and non-concurrent versions and then compare output
		f := make([]byte, l)
		g := make([]byte, l)
		h := make([]byte, l)
		if _, err := prng.Read(f); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(h); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(g, f)
		Xor(f, h)
		inPlaceXorBytes(g, h)
		for i := range f {
			if f[i] != g[i] {
				t.Fatalf("result of concurrent XOR operation did not match with result of non-concurrent equivalent")
			}
		}
	}
}

func TestConcurrentInPlaceAndBytes(t *testing.T) {
	for _, l := range []int{10, 16384, 10000000} {
		// XOR with itself
		a := make([]byte, l)
		b := make([]byte, l)
		if _, err := prng.Read(a); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(b, a)
		concurrentInPlaceAndBytes(a, a)
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("AND operation was not performed correctly")
			}
		}
		// AND same original slice with concurrent and non-concurrent versions and then compare output
		f := make([]byte, l)
		g := make([]byte, l)
		h := make([]byte, l)
		if _, err := prng.Read(f); err != nil {
			t.Fatalf("error generating random bytes")
		}
		if _, err := prng.Read(h); err != nil {
			t.Fatalf("error generating random bytes")
		}
		copy(g, f)
		concurrentInPlaceAndBytes(f, h)
		inPlaceAndBytes(g, h)
		for i := range f {
			if f[i] != g[i] {
				t.Fatalf("result of concurrent AND operation did not match with result of non-concurrent equivalent")
			}
		}
	}

}

func BenchmarkNaiveXorBytes(b *testing.B) {
	a := make([]byte, 10000000)
	prng.Read(a)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		naiveXorBytes(a, a)
	}
}

func BenchmarkInPlaceXorBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inPlaceXorBytes(a, a)
	}
}

func BenchmarkUnsafeInPlaceXorBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Xor(a, a)
	}
}

func BenchmarkConcurrentInPlaceXorBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		concurrentInPlaceXorBytes(a, a)
	}
}

func BenchmarkConcurrentUnsafeBitOperation(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConcurrentBitOp(Xor, a, a)
	}
}

func BenchmarkInPlaceAndBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inPlaceAndBytes(a, a)
	}
}

func BenchmarkConcurrentInPlaceAndBytes(b *testing.B) {
	a := make([]byte, 10000000)
	if _, err := prng.Read(a); err != nil {
		b.Fatalf("error generating random bytes")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		concurrentInPlaceAndBytes(a, a)
	}
}
