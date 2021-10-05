package kkrtpsi

import (
	"encoding/binary"
	"io"
	"runtime"
	"sync"

	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/oprf"
)

const (
	batchSize = 2048
)

// findK returns the number of base OT for OPRF
// these numbers are from the paper: Efficient Batched Oblivious PRF with Applications to Private Set Intersection
// by Vladimir Kolesnikov, Ranjit Kumaresan, Mike Rosulek, and Ni Treu in 2016.
// Reference:	http://dx.doi.org/10.1145/2976749.2978381 (KKRT)
func findK(n int64) int {
	switch {
	// 2^8
	case n > 0 && n <= 256:
		return 424
	// 2^12
	case n > 256 && n <= 4096:
		return 432
	// 2^16
	case n > 4096 && n <= 65536:
		return 440
	// 2^20
	case n > 65536:
		return 448
	default:
		return 128
	}
}

// HashRead reads one hash
func EncodesRead(r io.Reader, u *[cuckoo.Nhash]uint64) (err error) {
	err = binary.Read(r, binary.BigEndian, u)
	return
}

// HashWrite writes one hash out
func EncodesWrite(w io.Writer, u [cuckoo.Nhash]uint64) error {
	return binary.Write(w, binary.BigEndian, u)
}

type hashEncodingJob struct {
	batchSize   int
	identifiers []hashable
	hashed      [][cuckoo.Nhash]uint64
	execute     func(job hashEncodingJob)
}

func makeJob(hasher hash.Hasher, batchSize int, f func(hashEncodingJob)) hashEncodingJob {
	return hashEncodingJob{
		batchSize:   batchSize,
		identifiers: make([]hashable, batchSize),
		execute:     f}
}

func (id hashable) encodeAndHash(oprfKeys []oprf.Key, hasher hash.Hasher) (hashes [cuckoo.Nhash]uint64) {
	for hIdx, bucketIdx := range id.bucketIdx {
		encoded, _ := oprfKeys[bucketIdx].Encode(append(id.identifier, uint8(hIdx)))
		hashes[hIdx] = hasher.Hash64(encoded)
	}

	return
}

// HashAllParallel reads all identifiers from identifiers
// and parallel hashes them until identifiers closes
func EncodeAndHashAllParallel(oprfKeys []oprf.Key, hasher hash.Hasher, identifiers <-chan hashable) <-chan [cuckoo.Nhash]uint64 {
	// one wg.Add() per batch + one for the batcher go routine
	var jobPool = make(chan hashEncodingJob)
	var wg sync.WaitGroup

	// work on the jobPool
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func() {
			for job := range jobPool {
				var hashed = make([][cuckoo.Nhash]uint64, job.batchSize)
				for i := 0; i < job.batchSize; i++ {
					hashed[i] = job.identifiers[i].encodeAndHash(oprfKeys, hasher)
				}
				job.hashed = hashed
				job.execute(job)
			}
		}()
	}

	var encoded = make(chan [cuckoo.Nhash]uint64)
	execute := func(job hashEncodingJob) {
		// pump everything out
		for i := 0; i < job.batchSize; i++ {
			encoded <- job.hashed[i]
		}
		wg.Done()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		var i = 0
		// init a first batch
		var batch = makeJob(hasher, batchSize, execute)
		for identifier := range identifiers {
			// accumulate a batch
			batch.identifiers[i] = identifier
			i++
			// send it out?
			if i == batchSize {
				wg.Add(1)
				jobPool <- batch
				// reset batch
				batch = makeJob(hasher, batchSize, execute)
				i = 0
			}
		}
		// anything left here?
		if i != 0 {
			batch.batchSize = i
			wg.Add(1)
			jobPool <- batch
		}
	}()

	// turn the lights off on your way out
	go func() {
		wg.Wait()
		close(encoded)
	}()

	return encoded
}
