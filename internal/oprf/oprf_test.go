package oprf

import (
	"bytes"
	"crypto/aes"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/ot"
)

var (
	network   = "tcp"
	address   = "127.0.0.1:"
	baseCount = 1 << 16
	prng      = rand.New(rand.NewSource(time.Now().UnixNano()))
	choices   = genChoiceString()
)

func genChoiceString() [][]byte {
	choices := make([][]byte, baseCount)
	for i := range choices {
		choices[i] = make([]byte, 64)
		prng.Read(choices[i])
	}
	return choices
}

func makeCuckoo(choices [][]byte, seeds [cuckoo.Nhash][]byte) (*cuckoo.Cuckoo, error) {
	c := cuckoo.NewCuckoo(uint64(baseCount), seeds)
	in := make(chan []byte, baseCount)
	for _, id := range choices {
		in <- id
	}
	close(in)
	err := c.Insert(in)
	return c, err
}

func initOPRFReceiver(oprf OPRF, choices *cuckoo.Cuckoo, outBus chan<- [cuckoo.Nhash]map[uint64]uint64, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}

		go oprfReceiveHandler(conn, oprf, choices, outBus, errs)
	}()
	return l.Addr().String(), nil
}

func oprfReceiveHandler(conn net.Conn, oprf OPRF, choices *cuckoo.Cuckoo, outBus chan<- [cuckoo.Nhash]map[uint64]uint64, errs chan<- error) {
	defer close(outBus)

	out, err := oprf.Receive(choices, conn)
	if err != nil {
		errs <- err
	}

	outBus <- out
}

func testEncodings(encodedHashMap [cuckoo.Nhash]map[uint64]uint64, keys Key, seeds [cuckoo.Nhash][]byte, choicesCuckoo *cuckoo.Cuckoo) error {
	// Testing encodings
	senderCuckoo := cuckoo.NewDummyCuckoo(uint64(baseCount), seeds)
	hasher := senderCuckoo.GetHasher()
	var hashes [cuckoo.Nhash]uint64
	for i, id := range choices {
		// compute encoding and hash
		for hIdx, bIdx := range senderCuckoo.BucketIndices(id) {
			encoded, err := keys.Encode(bIdx, id, uint8(hIdx))
			if err != nil {
				return err
			}

			hashes[hIdx] = hasher.Hash64(encoded)
		}

		// test hashes
		var found bool
		for hIdx, hashed := range hashes {
			if idx, ok := encodedHashMap[hIdx][hashed]; ok {
				found = true
				id, _ := choicesCuckoo.GetItemWithHash(idx)
				if id == nil {
					return fmt.Errorf("failed to retrieve item #%v", idx)
				}

				if !bytes.Equal(id, choices[i]) {
					return fmt.Errorf("oprf failed, got: %v, want %v", id, choices[i])
				}
			}
		}

		if !found {
			return fmt.Errorf("failed to find proper encoding.")
		}
	}

	return nil
}

func TestKKRT(t *testing.T) {
	outBus := make(chan [cuckoo.Nhash]map[uint64]uint64)
	keyBus := make(chan Key)
	errs := make(chan error)

	// start timer
	start := time.Now()
	var seeds [cuckoo.Nhash][]byte
	for i := range seeds {
		seeds[i] = make([]byte, hash.SaltLength)
		rand.Read(seeds[i])
	}
	choicesCuckoo, err := makeCuckoo(choices, seeds)
	if err != nil {
		t.Fatal(err)
	}

	receiverOPRF, err := NewOPRF(KKRT, int(choicesCuckoo.Len()), ot.NaorPinkas)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOPRFReceiver(receiverOPRF, choicesCuckoo, outBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	senderOPRF, err := NewOPRF(KKRT, int(choicesCuckoo.Len()), ot.NaorPinkas)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		defer close(errs)
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		if err != nil {
			errs <- fmt.Errorf("Error creating IKNP OT: %s", err)
		}

		defer close(keyBus)
		keys, err := senderOPRF.Send(conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(outBus)
		}

		keyBus <- keys

	}()

	// any errors?
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}

	// Receive keys
	keys := <-keyBus

	// Receive msg
	encodedHashMap := <-outBus

	// stop timer
	end := time.Now()
	t.Logf("Time taken for %d KKRT OPRF is: %v\n", baseCount, end.Sub(start))

	// Testing encodings
	err = testEncodings(encodedHashMap, keys, seeds, choicesCuckoo)
	if err != nil {
		t.Fatal(err)
	}
}

func TestImprovedKKRT(t *testing.T) {
	outBus := make(chan [cuckoo.Nhash]map[uint64]uint64)
	keyBus := make(chan Key)
	errs := make(chan error, 5)

	// start timer
	start := time.Now()
	var seeds [cuckoo.Nhash][]byte
	for i := range seeds {
		seeds[i] = make([]byte, hash.SaltLength)
		rand.Read(seeds[i])
	}
	choicesCuckoo, err := makeCuckoo(choices, seeds)
	if err != nil {
		t.Fatal(err)
	}

	receiverOPRF, err := NewOPRF(ImprvKKRT, int(choicesCuckoo.Len()), ot.NaorPinkas)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOPRFReceiver(receiverOPRF, choicesCuckoo, outBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	senderOPRF, err := NewOPRF(ImprvKKRT, int(choicesCuckoo.Len()), ot.NaorPinkas)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		defer close(errs)
		conn, err := net.Dial(network, addr)
		if err != nil {
			errs <- fmt.Errorf("Cannot dial: %s", err)
		}
		if err != nil {
			errs <- fmt.Errorf("Error creating IKNP OT: %s", err)
		}

		defer close(keyBus)
		keys, err := senderOPRF.Send(conn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(outBus)
		}

		keyBus <- keys
	}()

	// any errors?
	//errors?
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}
	// Receive keys
	keys := <-keyBus

	// Receive msg
	encodedHashMap := <-outBus

	// stop timer
	end := time.Now()
	t.Logf("Time taken for %d ImprovedKKRT OPRF is: %v\n", baseCount, end.Sub(start))

	// Testing encodings
	// Testing encodings
	err = testEncodings(encodedHashMap, keys, seeds, choicesCuckoo)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkEncode(b *testing.B) {
	sk := make([]byte, 16)
	s := make([]byte, 64)
	q := make([][]byte, 1)
	q[0] = make([]byte, 65)
	prng.Read(sk)
	prng.Read(s)
	prng.Read(q[0])
	aesBlock, _ := aes.NewCipher(sk)
	key := Key{block: aesBlock, s: s, q: q}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key.Encode(0, q[0], 0)
	}
}
