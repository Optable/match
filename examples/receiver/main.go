package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/optable/match/examples/format"
	"github.com/optable/match/internal/util"
	"github.com/optable/match/pkg/psi"
)

const (
	defaultProtocol       = "npsi"
	defaultPort           = ":6667"
	defaultSenderFileName = "receiver-ids.txt"
	defaultCommonFileName = "common-ids.txt"
)

func usage() {
	log.Printf("Usage: receiver [-proto protocol] [-p port] [-in file] [-out file] [-once false]\n")
	flag.PrintDefaults()
}

var out *string

func main() {
	var wg sync.WaitGroup
	var protocol = flag.String("proto", defaultProtocol, "the psi protocol (bpsi,npsi,dhpsi,kkrt)")
	var port = flag.String("p", defaultPort, "The receiver port")
	var file = flag.String("in", defaultSenderFileName, "A list of IDs terminated with a newline")
	out = flag.String("out", defaultCommonFileName, "A list of IDs that intersect between the receiver and the sender")
	var once = flag.Bool("once", false, "Exit after processing one receiver")
	var verbose = flag.Int("v", 0, "Verbosity level, default to -v 0 for info level messages, -v 1 for debug messages, and -v 2 for trace level message.")
	var showHelp = flag.Bool("h", false, "Show help message")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		format.ShowUsageAndExit(usage, 0)
	}

	// validate protocol
	var psiType psi.Protocol
	switch *protocol {
	case "bpsi":
		psiType = psi.ProtocolBPSI
	case "npsi":
		psiType = psi.ProtocolNPSI
	case "dhpsi":
		psiType = psi.ProtocolDHPSI
	case "kkrt":
		psiType = psi.ProtocolKKRTPSI
	default:
		psiType = psi.ProtocolUnsupported
	}

	log.Printf("operating with protocol %s", psiType)
	// fetch stdr logger
	mlog := format.GetLogger(*verbose)

	// open file
	f, err := os.Open(*file)
	format.ExitOnErr(mlog, err, "failed to open file")
	defer f.Close()

	// count lines
	log.Printf("counting lines in %s", *file)
	t := time.Now()
	n, err := util.Count(f)
	format.ExitOnErr(mlog, err, "failed to count")
	log.Printf("that took %v", time.Since(t))
	log.Printf("operating on %s with %d IDs", *file, n)

	// get a listener
	l, err := net.Listen("tcp", *port)
	format.ExitOnErr(mlog, err, "failed to listen on tcp port")
	log.Printf("receiver listening on %s", *port)
	for {
		if c, err := l.Accept(); err != nil {
			format.ExitOnErr(mlog, err, "failed to accept incoming connection")
		} else {
			log.Printf("handling sender %s", c.RemoteAddr())
			f, err := os.Open(*file)
			format.ExitOnErr(mlog, err, "failed to open file")
			// enable nagle
			switch v := c.(type) {
			// enable nagle
			case *net.TCPConn:
				v.SetNoDelay(false)
			}
			// make the receiver

			receiver, err := psi.NewReceiver(psiType, c)
			format.ExitOnErr(mlog, err, "failed to create receiver")
			// and hand it off
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer c.Close()
				handle(receiver, n, f, logr.NewContext(context.Background(), mlog))
				log.Printf("handled sender %s", c.RemoteAddr())
			}()

			if *once {
				wg.Wait()
				return
			}
		}
	}
}

func handle(r psi.Receiver, n int64, f io.ReadCloser, ctx context.Context) {
	defer f.Close()
	ids := util.Exhaust(n, f)
	logger := logr.FromContextOrDiscard(ctx)
	if i, err := r.Intersect(ctx, n, ids); err != nil {
		format.ExitOnErr(logger, err, "intersect failed")
	} else {
		// write memory usage to stderr
		format.MemUsageToStdErr(logger)
		// write out to common-ids.txt
		log.Printf("intersected %d IDs, writing out to %s", len(i), *out)
		if f, err := os.Create(*out); err == nil {
			defer f.Close()
			for _, id := range i {
				// and write it
				if _, err := f.Write(append(id, "\n"...)); err != nil {
					format.ExitOnErr(logger, err, "failed to write intersected ID to file")
				}
			}
		} else {
			format.ExitOnErr(logger, err, "failed to perform PSI")
		}
	}
}
