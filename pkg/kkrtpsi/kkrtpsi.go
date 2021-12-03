package kkrtpsi

import (
	"encoding/binary"
	"io"
	"math"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/oprf"
)

// HashRead reads one hash
func EncodingsRead(r io.Reader, u *[cuckoo.Nhash]uint64) error {
	return binary.Read(r, binary.BigEndian, u)
}

// HashWrite writes one hash out
func EncodingsWrite(w io.Writer, u [cuckoo.Nhash]uint64) error {
	return binary.Write(w, binary.BigEndian, u)
}

func (bytes oprfEncodedInputs) encodeAndHash(oprfKeys *oprf.Key, hasher hash.Hasher) (hashes [cuckoo.Nhash]uint64) {
	// oprfInput is instantiated at the required size
	for hIdx, bucketIdx := range bytes.bucketIdx {
		oprfKeys.Encode(bucketIdx, bytes.prcEncoded[hIdx])
		hashes[hIdx] = hasher.Hash64(bytes.prcEncoded[hIdx])
	}

	return hashes
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
