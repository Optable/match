package kkrtpsi

import (
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

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

func (id hashable) encodeAndHash(oprfKeys oprf.Key, hasher hash.Hasher) (hashes [cuckoo.Nhash]uint64) {
	// oprfInput is instantiated at the required size
	for hIdx, bucketIdx := range id.bucketIdx {
		encoded, _ := oprfKeys.Encode(bucketIdx, id.identifier, uint8(hIdx))
		hashes[hIdx] = hasher.Hash64(encoded)
	}

	return
}

// HashAllParallel reads all identifiers from identifiers
// and parallel hashes them until identifiers closes
func EncodeAndHashAllParallel(oprfKeys oprf.Key, hasher hash.Hasher, identifiers <-chan hashable) <-chan [cuckoo.Nhash]uint64 {
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

func printStageStats(stage string, prevTime, startTime time.Time, prevMem uint64) (time.Time, uint64) {
	fmt.Println("====================================")
	fmt.Println(stage)
	endTime := time.Now()
	fmt.Println("Time:\033[34;1m", endTime.Sub(prevTime), "\033[0m(cum", endTime.Sub(startTime), ")")
	var m runtime.MemStats
	runtime.ReadMemStats(&m) // https://cs.opensource.google/go/go/+/go1.17.1:src/runtime/mstats.go;l=107
	fmt.Printf("Total memory from OS: \033[31;1m%v MiB\033[0m\n", (m.Sys-prevMem)/(1024*1024))
	fmt.Printf("Heap Alloc: %v MiB\n", m.HeapAlloc/(1024*1024))
	fmt.Printf("Heap Sys: %v MiB\n", m.HeapSys/(1024*1024))           // estimate largest size heap has had
	fmt.Printf("Total Cum Alloc: %v MiB\n", m.TotalAlloc/(1024*1024)) // cumulative heap allocations
	fmt.Printf("Cum NumGC calls: \033[33;1m%v\033[0m\n", m.NumGC)
	return endTime, m.Sys
}
