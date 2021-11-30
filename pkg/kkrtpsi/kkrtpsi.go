package kkrtpsi

import (
	"encoding/binary"
	"io"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/oprf"
)

const batchSize = 2048

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
	batchSize int
	prc       []oprfEncodedInputs // PseudoRandom Code
	hashed    [][cuckoo.Nhash]uint64
	execute   func(job hashEncodingJob)
}

func makeJob(hasher hash.Hasher, batchSize int, f func(hashEncodingJob)) hashEncodingJob {
	return hashEncodingJob{
		batchSize: batchSize,
		prc:       make([]oprfEncodedInputs, batchSize),
		execute:   f}
}

func (bytes oprfEncodedInputs) encodeAndHash(oprfKeys *oprf.Key, hasher hash.Hasher) (hashes [cuckoo.Nhash]uint64, err error) {
	// oprfInput is instantiated at the required size
	for hIdx, bucketIdx := range bytes.bucketIdx {
		err = oprfKeys.Encode(bucketIdx, bytes.prcEncoded[hIdx])
		if err != nil {
			return hashes, err
		}
		hashes[hIdx] = hasher.Hash64(bytes.prcEncoded[hIdx])
	}

	return hashes, nil
}

// HashAllParallel reads all identifiers from identifiers
// and parallel hashes them until identifiers closes
func EncodeAndHashAllParallel(oprfKeys *oprf.Key, message inputsAndHasher) <-chan [cuckoo.Nhash]uint64 {
	// one wg.Add() per batch + one for the batcher go routine
	var jobPool = make(chan hashEncodingJob)
	var wg sync.WaitGroup

	// work on the jobPool
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func() {
			var err error
			for job := range jobPool {
				var hashed = make([][cuckoo.Nhash]uint64, job.batchSize)
				for i := 0; i < job.batchSize; i++ {
					hashed[i], err = job.prc[i].encodeAndHash(oprfKeys, message.hasher)
					if err != nil {
						panic(err)
					}
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
		var batch = makeJob(message.hasher, batchSize, execute)
		for _, identifier := range message.inputs {
			// accumulate a batch
			batch.prc[i] = identifier
			i++
			// send it out?
			if i == batchSize {
				wg.Add(1)
				jobPool <- batch
				// reset batch
				batch = makeJob(message.hasher, batchSize, execute)
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

func printStageStats(log logr.Logger, stage int, prevTime, startTime time.Time, prevMem uint64) (time.Time, uint64) {
	endTime := time.Now()
	log.V(2).Info("stats", "stage", stage, "time", time.Since(prevTime).String(), "cumulative time", time.Since(startTime).String())
	var m runtime.MemStats
	runtime.ReadMemStats(&m) // https://cs.opensource.google/go/go/+/go1.17.1:src/runtime/mstats.go;l=107
	log.V(2).Info("stats", "stage", stage, "total memory from OS (MiB)", math.Round(float64(m.Sys-prevMem)*100/(1024*1024))/100)
	log.V(2).Info("stats", "stage", stage, "cumulative GC calls", m.NumGC)
	return endTime, m.Sys
}
