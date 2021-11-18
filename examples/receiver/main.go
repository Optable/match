package main

import (
	"context"
	"flag"
	"io"
	"log"
	"math"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
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

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

func memUsageToStdErr(logger logr.Logger) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m) // https://cs.opensource.google/go/go/+/go1.17.1:src/runtime/mstats.go;l=107
	logger.V(1).Info("Final stats", "Total memory (GiB)", math.Round(float64(m.Sys)*100/(1024*1024*1024))/100)
	logger.V(1).Info("Final stats", "Garbage collector calls:", m.NumGC)
}

func exitOnErr(logger logr.Logger, err error, msg string) {
	if err != nil {
		logger.Error(err, msg)
		os.Exit(1)
	}
}

// getLogger returns a stdr.Logger that implements the logr.Logger interface
// and sets the verbosity of the returned logger.
// set v to 0 for info level messages,
// 1 for debug messages and 2 for trace level message.
// any other verbosity level will default to 0.
func getLogger(v int) logr.Logger {
	logger := stdr.New(nil)
	// bound check
	if v > 2 || v < 0 {
		v = 0
		logger.Info("Invalid verbosity, setting logger to display info level messages only.")
	}
	stdr.SetVerbosity(v)

	return logger
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
		showUsageAndExit(0)
	}

	// validate protocol
	var psiType psi.Protocol
	switch *protocol {
	case "bpsi":
		psiType = psi.BPSI
	case "npsi":
		psiType = psi.NPSI
	case "dhpsi":
		psiType = psi.DHPSI
	case "kkrt":
		psiType = psi.KKRTPSI
	default:
		log.Printf("unsupported protocol %s", *protocol)
		showUsageAndExit(0)
	}

	log.Printf("operating with protocol %s", *protocol)
	// fetch stdr logger
	mlog := getLogger(*verbose)

	// open file
	f, err := os.Open(*file)
	exitOnErr(mlog, err, "failed to open file")
	defer f.Close()

	// count lines
	log.Printf("counting lines in %s", *file)
	t := time.Now()
	n, err := util.Count(f)
	exitOnErr(mlog, err, "failed to count")
	log.Printf("that took %v", time.Since(t))
	log.Printf("operating on %s with %d IDs", *file, n)

	// get a listener
	l, err := net.Listen("tcp", *port)
	exitOnErr(mlog, err, "failed to listen on tcp port")
	log.Printf("receiver listening on %s", *port)
	for {
		if c, err := l.Accept(); err != nil {
			exitOnErr(mlog, err, "failed to accept incoming connection")
		} else {
			log.Printf("handling sender %s", c.RemoteAddr())
			f, err := os.Open(*file)
			exitOnErr(mlog, err, "failed to open file")
			// enable nagle
			switch v := c.(type) {
			// enable nagle
			case *net.TCPConn:
				v.SetNoDelay(false)
			}
			// make the receiver
			receiver, _ := psi.NewReceiver(psiType, c)
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
		exitOnErr(logger, err, "intersect failed")
	} else {
		// write memory usage to stderr
		memUsageToStdErr(logger)
		// write out to common-ids.txt
		log.Printf("intersected %d IDs, writing out to %s", len(i), *out)
		if f, err := os.Create(*out); err == nil {
			defer f.Close()
			for _, id := range i {
				// and write it
				if _, err := f.Write(append(id, "\n"...)); err != nil {
					exitOnErr(logger, err, "failed to write intersected ID to file")
				}
			}
		} else {
			exitOnErr(logger, err, "failed to perform PSI")
		}
	}
}
