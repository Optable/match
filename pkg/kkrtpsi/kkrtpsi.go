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

// HashRead reads one hash
func EncodesRead(r io.Reader, u *[cuckoo.Nhash]uint64) (err error) {
	err = binary.Read(r, binary.BigEndian, u)
	return
}

// HashWrite writes one hash out
func EncodesWrite(w io.Writer, u [cuckoo.Nhash]uint64) error {
	return binary.Write(w, binary.BigEndian, u)
}

func (bytes oprfEncodedInputs) encodeAndHash(oprfKeys *oprf.Key, hasher hash.Hasher) (hashes [cuckoo.Nhash]uint64, err error) {
	// oprfInput is instantiated at the required size
	for hIdx, bucketIdx := range bytes.bucketIdx {
		oprfKeys.Encode(bucketIdx, bytes.prcEncoded[hIdx])
		hashes[hIdx] = hasher.Hash64(bytes.prcEncoded[hIdx])
	}

	return hashes, nil
}

// HashAllParallel accepts the inputsAndHasher struct
// which contains the identifiers and the hasher and
// parallel hashes them until identifiers closes
func EncodeAndHashAllParallel(oprfKeys *oprf.Key, message inputsAndHasher) <-chan [cuckoo.Nhash]uint64 {
	nworkers := runtime.GOMAXPROCS(0)
	var encoded = make(chan [cuckoo.Nhash]uint64, 1024)

	// determine number of blocks to split original matrix
	workerResp := len(message.inputs) / nworkers

	// Run a worker pool
	var wg sync.WaitGroup
	wg.Add(nworkers)
	for w := 0; w < nworkers; w++ {
		w := w
		go func() {
			defer wg.Done()
			step := workerResp * w
			if w == nworkers-1 { // last block
				for i := step; i < len(message.inputs); i++ {
					hashes, err := message.inputs[i].encodeAndHash(oprfKeys, message.hasher)
					if err != nil {
						panic(err)
					}
					encoded <- hashes
				}
			} else {
				for i := step; i < step+workerResp; i++ {
					hashes, err := message.inputs[i].encodeAndHash(oprfKeys, message.hasher)
					if err != nil {
						panic(err)
					}
					encoded <- hashes
				}
			}
		}()
	}

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
