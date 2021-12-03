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

const msgCount = 1 << 16

func genChoiceString() [][]byte {
	choices := make([][]byte, msgCount)
	for i := range choices {
		choices[i] = make([]byte, 66)
		rand.Read(choices[i])
	}
	return choices
}

func makeCuckoo(choices [][]byte, seeds [cuckoo.Nhash][]byte) (*cuckoo.Cuckoo, error) {
	c := cuckoo.NewCuckoo(uint64(msgCount), seeds)
	for _, id := range choices {
		if err := c.Insert(id); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func testEncodings(encodedHashMap []map[uint64]uint64, key *Key, sk []byte, seeds [cuckoo.Nhash][]byte, choicesCuckoo *cuckoo.Cuckoo, choices [][]byte) error {
	senderCuckoo := cuckoo.NewCuckooHasher(uint64(msgCount), seeds)
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
			key.Encode(bIdx, pseudorandId)
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

func TestOPRF(t *testing.T) {
	outBus := make(chan []map[uint64]uint64, cuckoo.Nhash)
	keyBus := make(chan *Key)
	errs := make(chan error, 1)
	sk := make([]byte, 16)
	choices := genChoiceString()

	// start timer
	start := time.Now()
	// sample seeds
	var seeds [cuckoo.Nhash][]byte
	for i := range seeds {
		seeds[i] = make([]byte, hash.SaltLength)
		rand.Read(seeds[i])
	}

	// generate oprf Input
	choicesCuckoo, err := makeCuckoo(choices, seeds)
	if err != nil {
		t.Fatal(err)
	}
	oprfInputSize := int(choicesCuckoo.Len())

	// generate AES secret key (16-byte)
	if _, err := rand.Read(sk); err != nil {
		t.Fatal(err)
	}

	// create client, server connections
	senderConn, receiverConn := net.Pipe()

	// sender
	go func() {
		defer close(errs)
		defer close(keyBus)
		keys, err := NewOPRF(oprfInputSize).Send(senderConn)
		if err != nil {
			errs <- fmt.Errorf("Send encountered error: %s", err)
			close(outBus)
		}

		keyBus <- keys
	}()

	// receiver
	go func() {
		defer close(outBus)
		out, err := NewOPRF(oprfInputSize).Receive(choicesCuckoo, sk, receiverConn)
		if err != nil {
			errs <- err
		}
		outBus <- out
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
	t.Logf("Time taken for %d OPRF is: %v\n", msgCount, end.Sub(start))

	// Testing encodings
	err = testEncodings(encodedHashMap, keys, sk, seeds, choicesCuckoo, choices)
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
	key := Key{secret: s, oprfKeys: q}
	bytes := crypto.PseudorandomCode(aesBlock, s, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key.Encode(0, bytes)
	}
}
