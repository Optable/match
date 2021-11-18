package oprf

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
)

var (
	network   = "tcp"
	address   = "127.0.0.1:"
	baseCount = 1 << 16
	choices   = genChoiceString()
)

func genChoiceString() [][]byte {
	choices := make([][]byte, baseCount)
	for i := range choices {
		choices[i] = make([]byte, 64)
		rand.Read(choices[i])
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

func initOPRFReceiver(oprf OPRF, choices *cuckoo.Cuckoo, sk []byte, outBus chan<- [cuckoo.Nhash]map[uint64]uint64, errs chan<- error) (string, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		errs <- fmt.Errorf("net listen encountered error: %s", err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			errs <- fmt.Errorf("Cannot create connection in listen accept: %s", err)
		}

		go oprfReceiveHandler(conn, oprf, choices, sk, outBus, errs)
	}()
	return l.Addr().String(), nil
}

func oprfReceiveHandler(conn net.Conn, oprf OPRF, choices *cuckoo.Cuckoo, sk []byte, outBus chan<- [cuckoo.Nhash]map[uint64]uint64, errs chan<- error) {
	defer close(outBus)

	out, err := oprf.Receive(choices, sk, conn)
	if err != nil {
		errs <- err
	}

	outBus <- out
}

func testEncodings(encodedHashMap [cuckoo.Nhash]map[uint64]uint64, keys Key, sk []byte, seeds [cuckoo.Nhash][]byte, choicesCuckoo *cuckoo.Cuckoo) error {
	// Testing encodings
	senderCuckoo := cuckoo.NewDummyCuckoo(uint64(baseCount), seeds)
	hasher := senderCuckoo.GetHasher()
	var hashes [cuckoo.Nhash]uint64

	aesBlock, err := aes.NewCipher(sk)
	if err != nil {
		return err
	}
	for i, id := range choices {
		// compute encoding and hash
		for hIdx, bIdx := range senderCuckoo.BucketIndices(id) {
			pseudorandId := crypto.PseudorandomCode(aesBlock, id, byte(hIdx))
			err = keys.Encode(bIdx, pseudorandId)
			if err != nil {
				return err
			}

			hashes[hIdx] = hasher.Hash64(pseudorandId)
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

func TestImprovedKKRT(t *testing.T) {
	outBus := make(chan [cuckoo.Nhash]map[uint64]uint64)
	keyBus := make(chan Key)
	errs := make(chan error, 5)
	sk := make([]byte, 16)

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
	// generate AES secret key (16-byte)
	rand.Read(sk)

	receiverOPRF, err := NewOPRF(int(choicesCuckoo.Len()))
	if err != nil {
		t.Fatal(err)
	}

	addr, err := initOPRFReceiver(receiverOPRF, choicesCuckoo, sk, outBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	senderOPRF, err := NewOPRF(int(choicesCuckoo.Len()))
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
	err = testEncodings(encodedHashMap, keys, sk, seeds, choicesCuckoo)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkEncode(b *testing.B) {
	sk := make([]byte, 16)
	s := make([]byte, 64)
	q := make([][]byte, 1)
	q[0] = make([]byte, 64)
	rand.Read(sk)
	rand.Read(s)
	rand.Read(q[0])
	aesBlock, err := aes.NewCipher(sk)
	if err != nil {
		b.Fatal(err)
	}
	key := Key{s: s, q: q}
	bytes := crypto.PseudorandomCode(aesBlock, s, 0)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := key.Encode(0, bytes); err != nil {
			b.Fatal(err)
		}
	}
}
