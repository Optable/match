package ot

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/bits-and-blooms/bitset"
	"github.com/optable/match/internal/util"
)

var (
	network        = "tcp"
	address        = "127.0.0.1:"
	curve          = "P256"
	cipherMode     = XORBlake3
	baseCount      = 102400
	messages       = genMsg(baseCount, 2)
	bitsetMessages = genBitSetMsg(baseCount, 2)
	msgLen         = make([]int, len(messages))
	choices        = genChoiceBits(baseCount)
	bitsetChoices  = genChoiceBitSet(baseCount)
	r              = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func genMsg(n, t int) [][][]byte {
	data := make([][][]byte, n)
	for i := 0; i < n; i++ {
		data[i] = make([][]byte, t)
		for j := range data[i] {
			data[i][j] = make([]byte, 64)
			r.Read(data[i][j])
		}
	}

	return data
}

func genBitSetMsg(n, t int) [][]*bitset.BitSet {
	data := make([][]*bitset.BitSet, n)
	for i := 0; i < n; i++ {
		data[i] = make([]*bitset.BitSet, t)
		for j := range data[i] {
			data[i][j] = util.SampleBitSetSlice(r, 64)
		}
	}

	return data
}

func genChoiceBitSet(n int) *bitset.BitSet {
	return util.SampleBitSetSlice(r, n)
}

func genChoiceBits(n int) []uint8 {
	choices := make([]uint8, n)
	util.SampleBitSlice(r, choices)
	return choices
}

func initReceiver(ot OT, choices []uint8, msgBus chan<- []byte, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("cannot create connection in listen accept: %s", err)
		}

		go receiveHandler(conn, ot, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func initBitSetReceiver(ot naorPinkas, choices []uint8, msgBus chan<- *bitset.BitSet, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("cannot create connection in listen accept: %s", err)
		}

		go receiveBitSetHandler(conn, ot, choices, msgBus, errs)
	}()
	return l.Addr().String(), nil
}

func receiveHandler(conn net.Conn, ot OT, choices []uint8, msgBus chan<- []byte, errs chan<- error) {
	defer close(msgBus)

	msg := make([][]byte, baseCount)
	err := ot.Receive(choices, msg, conn)
	if err != nil {
		errs <- err
	}

	for _, m := range msg {
		msgBus <- m
	}
}

func receiveBitSetHandler(conn net.Conn, ot naorPinkas, choices []uint8, msgBus chan<- *bitset.BitSet, errs chan<- error) {
	defer close(msgBus)

	//msg := make([][]byte, baseCount)
	msg := make([]*bitset.BitSet, baseCount)
	//err := ot.Receive(choices, msg, conn)
	err := ot.Receive(util.BitsToBitSet(choices), msg, conn)
	if err != nil {
		errs <- err
	}

	for _, m := range msg {
		msgBus <- m
	}
}

func testSimplestOT(t *testing.T) {
	for i, m := range messages {
		msgLen[i] = len(m[0])
	}

	msgBus := make(chan []byte)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()

	receiverOT, err := NewBaseOT(Simplest, false, baseCount, curve, msgLen, cipherMode)
	if err != nil {
		t.Fatalf("Error creating Simplest OT: %s", err)
	}

	addr, err := initReceiver(receiverOT, choices, msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		senderOT, err := NewBaseOT(Simplest, false, baseCount, curve, msgLen, cipherMode)
		if err != nil {
			errs <- fmt.Errorf("Error creating simplest OT: %s", err)
		}

		err = senderOT.Send(messages, conn)
		if err != nil {
			errs <- fmt.Errorf("send encountered error: %s", err)
			close(msgBus)
		}

	}()

	// Receive msg
	var msg [][]byte
	for m := range msgBus {
		msg = append(msg, m)
	}

	//errors?
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}

	// stop timer
	end := time.Now()
	t.Logf("Time taken for simplest OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		if !bytes.Equal(m, messages[i][choices[i]]) {
			t.Fatalf("OT failed got: %s, want %s", m, messages[i][choices[i]])
		}
	}
}

func TestNaorPinkasOT(t *testing.T) {
	for i, m := range bitsetMessages {
		msgLen[i] = int(m[0].Len())
	}

	msgBus := make(chan *bitset.BitSet)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()

	//ot, err := NewBaseOT(NaorPinkas, false, baseCount, curve, msgLen, cipherMode)
	ot, err := newNaorPinkas(baseCount, curve, msgLen, cipherMode)
	if err != nil {
		t.Fatalf("Error creating NaorPinkas OT: %s", err)
	}

	addr, err := initBitSetReceiver(ot, util.BitSetToBits(bitsetChoices), msgBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		ss, err := newNaorPinkas(baseCount, curve, msgLen, cipherMode)
		if err != nil {
			errs <- fmt.Errorf("Error creating simplest OT: %s", err)
		}

		err = ss.Send(bitsetMessages, conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(msgBus)
		}

	}()

	// Receive msg
	var msg []*bitset.BitSet
	for m := range msgBus {
		msg = append(msg, m)
	}

	//errors?
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}

	// stop timer
	end := time.Now()
	t.Logf("Time taken for NaorPinkas OT of %d OTs is: %v\n", baseCount, end.Sub(start))

	// verify if the received msgs are correct:
	if len(msg) == 0 {
		t.Fatal("OT failed, did not receive any messages")
	}

	for i, m := range msg {
		var choice uint8
		if bitsetChoices.Test(uint(i)) {
			choice = 1
		}
		if !m.Equal(bitsetMessages[i][choice]) {
			t.Fatalf("OT failed at message %d, got: %s, want %s", i, m, bitsetMessages[i][choice])
		}
	}
}

func benchmarkSampleBitSlice2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		util.SampleBitSlice(r, choices)
	}
}

func BenchmarkSampleBitSetSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		util.SampleBitSetSlice(r, baseCount)
	}
}

func BenchmarkTranspose(t *testing.B) {
	for i := 0; i < t.N; i++ {
		bm := util.BitSetsToBitMatrix(bitsetMessages[0])
		tm := util.Transpose(bm)
		ttm := util.Transpose(tm)
		for _, y := range tm {
			util.XorBytes(y, y)
		}
		bbm := util.BitMatrixToBitSets(ttm)
		for j, x := range bbm {
			if !x.Equal(bitsetMessages[0][j]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkContiguousTranspose(t *testing.B) {
	for i := 0; i < t.N; i++ {
		bm := util.BitSetsToBitMatrix(bitsetMessages[0])
		tm := util.ContiguousTranspose(bm)
		ttm := util.ContiguousTranspose(tm)
		for _, y := range tm {
			util.XorBytes(y, y)
		}
		bbm := util.BitMatrixToBitSets(ttm)
		for j, x := range bbm {
			if !x.Equal(bitsetMessages[0][j]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

/*
func BenchmarkContiguousTranspose2(t *testing.B) {
	for i := 0; i < t.N; i++ {
		bm := util.BitSetsToBitMatrix(bitsetMessages[0])
		tm := util.ContiguousTranspose2(bm)
		ttm := util.ContiguousTranspose2(tm)
		for _, y := range tm {
			util.XorBytes(y, y)
		}
		bbm := util.BitMatrixToBitSets(ttm)
		for j, x := range bbm {
			if !x.Equal(bitsetMessages[0][j]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkContiguousTranspose3(t *testing.B) {
	for i := 0; i < t.N; i++ {
		bm := util.BitSetsToBitMatrix(bitsetMessages[0])
		lm := util.Linearize2DMatrix(bm)
		tm := util.ContiguousTranspose3(lm, len(bm[0]), len(bm))
		ttm := util.ContiguousTranspose3(tm, len(bm), len(bm[0]))
		dm := util.Reconstruct2DMatrix(ttm, len(bm[0]))
		for _, y := range dm {
			util.XorBytes(y, y)
		}
		bbm := util.BitMatrixToBitSets(dm)
		for j, x := range bbm {
			if !x.Equal(bitsetMessages[0][j]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkContiguousTranspose4(t *testing.B) {
	for i := 0; i < t.N; i++ {
		orig := int(bitsetMessages[0][0].Len())
		tran := len(bitsetMessages[0])
		bm := util.BitSetsToBitSlice(bitsetMessages[0])
		tm := util.ContiguousTranspose3(bm, orig, tran)
		ttm := util.ContiguousTranspose3(tm, tran, orig)
		util.XorBytes(ttm, ttm)
		bbm := util.BitSliceToBitSets(ttm, orig)
		for j, x := range bbm {
			if !x.Equal(bitsetMessages[0][j]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkContiguousParallelTranspose(t *testing.B) {
	for i := 0; i < t.N; i++ {
		orig := int(bitsetMessages[0][0].Len())
		tran := len(bitsetMessages[0])
		bm := util.BitSetsToBitSlice(bitsetMessages[0])
		tm := make([]uint8, orig*tran)
		var wg sync.WaitGroup
		wg.Add(tran)
		for j := 0; j < orig*tran; j += orig {
			go util.ContiguousParallelTranspose(bm, tm, j, orig, tran, &wg)
		}
		wg.Wait()
		wg.Add(orig)
		ttm := make([]uint8, tran*orig)
		for k := 0; k < tran*orig; k += tran {
			go util.ContiguousParallelTranspose(tm, ttm, k, tran, orig, &wg)
		}
		wg.Wait()
		util.XorBytes(ttm, ttm)
		bbm := util.BitSliceToBitSets(ttm, orig)
		for k, x := range bbm {
			if !x.Equal(bitsetMessages[0][k]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkContiguousParallelTranspose2(t *testing.B) {
	for i := 0; i < t.N; i++ {
		orig := int(bitsetMessages[0][0].Len())
		tran := len(bitsetMessages[0])
		bm := util.BitSetsToBitSlice(bitsetMessages[0])
		tm := util.ContiguousParallelTranspose2(bm, orig, tran)
		ttm := util.ContiguousParallelTranspose2(tm, tran, orig)
		util.XorBytes(ttm, ttm)
		bbm := util.BitSliceToBitSets(ttm, orig)
		for k, x := range bbm {
			if !x.Equal(bitsetMessages[0][k]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkContiguousParallelTranspose3(t *testing.B) {
	for i := 0; i < t.N; i++ {
		orig := len(bitsetMessages[0][0].Bytes()) * 8
		tran := len(bitsetMessages[0])
		bm := util.BitSetsToBitSlice(bitsetMessages[0])
		tm := util.ContiguousParallelTranspose3(bm, orig, tran)
		ttm := util.ContiguousParallelTranspose3(tm, tran, orig)
		util.XorBytes(ttm, ttm)
	}
}

func BenchmarkTransposeBitSet(t *testing.B) {
	for i := 0; i < t.N; i++ {
		tm := util.TransposeBitSets(bitsetMessages[0])
		ttm := util.TransposeBitSets(tm)
		for _, y := range tm {
			util.XorBitsets(y, y)
		}
		for j, x := range ttm {
			if !x.Equal(bitsetMessages[0][j]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkTransposeBitSet2(t *testing.B) {
	for i := 0; i < t.N; i++ {
		tm := util.TransposeBitSets2(bitsetMessages[0])
		ttm := util.TransposeBitSets2(tm)
		for _, y := range tm {
			util.XorBitsets(y, y)
		}
		for j, x := range ttm {
			if !x.Equal(bitsetMessages[0][j]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkContiguousBitSetTranspose(t *testing.B) {
	for i := 0; i < t.N; i++ {
		tm := util.ContiguousBitSetTranspose(bitsetMessages[0])
		ttm := util.ContiguousBitSetTranspose(tm)
		for _, y := range tm {
			util.XorBitsets(y, y)
		}
		for j, x := range ttm {
			if !x.Equal(bitsetMessages[0][j]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkContiguousBitSetTranspose2(t *testing.B) {
	for i := 0; i < t.N; i++ {
		tm := util.ContiguousBitSetTranspose2(bitsetMessages[0])
		ttm := util.ContiguousBitSetTranspose2(tm)
		for _, y := range tm {
			util.XorBitsets(y, y)
		}
		for j, x := range ttm {
			if !x.Equal(bitsetMessages[0][j]) {
				t.Fatalf("Transpose failed. Doubly transposed message did not match original.")
			}
		}
	}
}

func BenchmarkExpandBitSets(t *testing.B) {
	a := util.SampleRandomBitSetMatrix(r, 4035, 12689)
	for i := 0; i < t.N; i++ {
		util.ExpandBitSets(a)
	}
}

// expand BitSet matrix using the underlying uint64s
func BenchmarkExpandBitSetInts(t *testing.B) {
	a := util.SampleRandomBitSetMatrix(r, 4035, 12689)
	for i := 0; i < t.N; i++ {
		util.ExpandBitSetInts(a)
	}
}
*/
