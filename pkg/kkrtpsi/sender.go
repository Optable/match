package kkrtpsi

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	"github.com/optable/match/internal/crypto"
	"github.com/optable/match/internal/cuckoo"
	"github.com/optable/match/internal/hash"
	"github.com/optable/match/internal/oprf"
	"github.com/optable/match/internal/util"
	"golang.org/x/sync/errgroup"
)

// stage 1: samples cuckoo.Nhash hash seeds and sends them to receiver for cuckoo hash
// stage 2: act as sender in OPRF, and receive OPRF keys
// stage 3: compute OPRF(k, id) and send them to receiver for intersection.

// Sender side of the KKRTPSI protocol
type Sender struct {
	rw io.ReadWriter
}

// inputToOprfEncode stores the possible bucket
// indexes in the receiver cuckoo hash table
type inputToOprfEncode struct {
	prcEncoded [cuckoo.Nhash][]byte // PseudoRandom Code
	bucketIdx  [cuckoo.Nhash]uint64
}

// stage1Result is used to pass the OPRF encoded
// inputs along with the hasher from stage 1 to stage 3
type stage1Result struct {
	inputs []inputToOprfEncode
	hasher hash.Hasher
}

// NewSender returns a KKRTPSI sender initialized to
// use rw as the communication layer
func NewSender(rw io.ReadWriter) *Sender {
	return &Sender{rw: rw}
}

// Send initiates a KKRTPSI exchange
// that reads local IDs from identifiers, until identifiers closes.
// The format of an indentifier is string
// example:
//  0e1f461bbefa6e07cc2ef06b9ee1ed25101e24d4345af266ed2f5a58bcd26c5e
func (s *Sender) Send(ctx context.Context, n int64, identifiers <-chan []byte) (err error) {
	// fetch and set up logger
	logger := logr.FromContextOrDiscard(ctx)
	logger = logger.WithValues("protocol", "kkrtpsi")

	// statistics
	start := time.Now()
	timer := time.Now()
	var mem uint64

	var seeds [cuckoo.Nhash][]byte
	var remoteN int64     // receiver size
	var oprfInputSize int // nb of OPRF keys

	var oprfKey *oprf.Key
	var encodedInputChan = make(chan stage1Result)

	// stage 1: sample hash seeds and write them to receiver
	// for cuckoo hashing parameters agreement.
	// read local ids and store the potential bucket indexes for each id.
	stage1 := func() error {
		logger.V(1).Info("Starting stage 1")

		// sample cuckoo.Nhash hash seeds
		for i := range seeds {
			seeds[i] = make([]byte, hash.SaltLength)
			if _, err := rand.Read(seeds[i]); err != nil {
				return err
			}
			// write it into rw
			if _, err := s.rw.Write(seeds[i]); err != nil {
				return err
			}
		}

		// read remote input size
		if err := binary.Read(s.rw, binary.BigEndian, &remoteN); err != nil {
			return err
		}

		// sample random 16 byte secret key for AES-128 and send to the receiver
		secretKey := make([]byte, aes.BlockSize)
		if _, err = rand.Read(secretKey); err != nil {
			return err
		}

		// send the secret key
		if _, err := s.rw.Write(secretKey); err != nil {
			return err
		}

		// calculate number of OPRF from the receiver based on
		// number of buckets in cuckooHashTable
		oprfInputSize = int(cuckoo.Factor * float64(remoteN))
		if 1 > oprfInputSize {
			oprfInputSize = 1
		}

		// instantiate an AES block
		aesBlock, err := aes.NewCipher(secretKey)
		if err != nil {
			return err
		}

		// exhaust local ids, and precompute all potential
		// hashes and store them using the same
		// cuckoo hash table parameters as the receiver.
		go func() {
			cuckooHasher := cuckoo.NewCuckooHasher(uint64(remoteN), seeds)

			// prepare struct to send inputs and hasher to stage 3
			var result stage1Result
			result.inputs = make([]inputToOprfEncode, n)

			var i int
			for id := range identifiers {
				// hash and calculate pseudorandom code given each possible hash index
				var bytes [cuckoo.Nhash][]byte
				for hIdx := 0; hIdx < cuckoo.Nhash; hIdx++ {
					bytes[hIdx] = crypto.PseudorandomCode(aesBlock, id, byte(hIdx))
				}
				result.inputs[i] = inputToOprfEncode{prcEncoded: bytes, bucketIdx: cuckooHasher.BucketIndices(id)}
				i++
			}

			result.hasher = cuckooHasher.GetHasher()
			encodedInputChan <- result
		}()

		// end stage1
		timer, mem = printStageStats(logger, 1, start, start, 0)
		logger.V(1).Info("Finished stage 1")
		return nil
	}

	// stage 2: act as sender in OPRF, and receive OPRF keys
	stage2 := func() error {
		logger.V(1).Info("Starting stage 2")

		// instantiate OPRF sender with agreed parameters
		oprfKey, err = oprf.NewOPRF(oprfInputSize).Send(s.rw)
		if err != nil {
			return err
		}

		// end stage2
		timer, mem = printStageStats(logger, 2, timer, start, mem)
		logger.V(1).Info("Finished stage 2")
		return nil
	}

	// stage 3: compute all possible OPRF output using keys obtained from stage2
	stage3 := func() error {
		logger.V(1).Info("Starting stage 3")

		// inform the receiver the number of local ID
		if err := binary.Write(s.rw, binary.BigEndian, &n); err != nil {
			return err
		}

		message := <-encodedInputChan
		nWorkers := runtime.GOMAXPROCS(0)
		var localEncodings = make(chan [][cuckoo.Nhash]uint64, nWorkers*2)

		batchSize := 2048
		nBatches := len(message.inputs) / batchSize
		workerResp := nBatches / nWorkers

		g, ctx := errgroup.WithContext(ctx)

		// each worker is responsible to encode and hash workerResp batches and send them out
		for w := 0; w < nWorkers; w++ {
			w := w
			g.Go(func() error {
				for batchNumber := 0; batchNumber < workerResp; batchNumber++ {
					batch := make([][cuckoo.Nhash]uint64, batchSize)
					step := (w*workerResp + batchNumber) * batchSize
					for bIdx := 0; bIdx < batchSize; bIdx++ {
						batch[bIdx] = message.inputs[step+bIdx].encodeAndHash(oprfKey, message.hasher)
					}

					select {
					case <-ctx.Done():
						return ctx.Err()
					// batch is filled; send it out
					case localEncodings <- batch:
					}
				}
				return nil
			})
		}

		// Extra worker deals with the remaining inputs
		// In the case that nBatches < nWorkers, this worker will be responsible
		// for the entire set of inputs. In addition, since it aggregates its
		// inputs into a single large batch, this process is not pipelined.
		g.Go(func() error {
			workLeft := len(message.inputs) - (workerResp * nWorkers * batchSize)
			if workLeft == 0 {
				return nil
			}

			lastBatch := make([][cuckoo.Nhash]uint64, workLeft)
			for bIdx := 0; bIdx < workLeft; bIdx++ {
				lastBatch[bIdx] = message.inputs[len(message.inputs)-workLeft+bIdx].encodeAndHash(oprfKey, message.hasher)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			// final batch filled; send it out
			case localEncodings <- lastBatch:
				return nil
			}
		})

		g.Go(func() error {
			// Add a buffer of 64k to amortize syscalls cost
			var bufferedWriter = bufio.NewWriterSize(s.rw, 1024*64)
			defer bufferedWriter.Flush()
			var sent int
			// no message
			if len(message.inputs) == 0 {
				close(localEncodings)
			}

			for batch := range localEncodings {
				for _, hashedEncodings := range batch {
					// send all encodings of an ID at once
					if err := EncodingsWrite(bufferedWriter, hashedEncodings); err != nil {
						return fmt.Errorf("stage3: %v", err)
					}
					sent++
				}
				if sent == len(message.inputs) {
					close(localEncodings)
				}
			}
			return nil
		})

		if err := g.Wait(); err != nil {
			return err
		}

		// end stage3
		_, _ = printStageStats(logger, 3, timer, start, mem)
		logger.V(1).Info("Finished stage 3")
		return nil
	}

	// run stage1
	if err := util.Sel(ctx, stage1); err != nil {
		return err
	}

	// run stage2
	if err := util.Sel(ctx, stage2); err != nil {
		return err
	}

	// run stage3
	if err := util.Sel(ctx, stage3); err != nil {
		return err
	}

	return nil
}
