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

	"github.com/optable/match/internal/util"
	"github.com/optable/match/pkg/psi"
	matchlog "github.com/optable/match/pkg/log"
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

var out *string
func main() {
	var wg sync.WaitGroup
	var protocol = flag.String("proto", defaultProtocol, "the psi protocol (dhpsi,npsi)")
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
	default:
		log.Printf("unsupported protocol %s", *protocol)
		showUsageAndExit(0)
	}

	log.Printf("operating with protocol %s", *protocol)
	// fetch stdr logger
	mlog := matchlog.GetLogger(*verbose)
	

	// open file
	f, err := os.Open(*file)
	if err != nil {
		mlog.Error(err, "failed to open file")
		os.Exit(1)
	}
	defer f.Close()

	// count lines
	log.Printf("counting lines in %s", *file)
	t := time.Now()
	n, err := util.Count(f)
	if err != nil {
		mlog.Error(err, "failed to count")
		os.Exit(1)
	}
	log.Printf("that took %v", time.Since(t))
	log.Printf("operating on %s with %d IDs", *file, n)

	// get a listener
	l, err := net.Listen("tcp", *port)
	if err != nil {
		mlog.Error(err, "failed to listen on tcp port: %v", *port)
		os.Exit(1)
	}
	log.Printf("receiver listening on %s", *port)
	for {
		if c, err := l.Accept(); err != nil {
			mlog.Error(err, "failed to accept incoming connection")
			os.Exit(1)
		} else {
			log.Printf("handling sender %s", c.RemoteAddr())
			f, err := os.Open(*file)
			if err != nil {
				mlog.Error(err, "failed to open file")
				os.Exit(1)
			}
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
				ctx := matchlog.ContextWithLogger(context.Background(), mlog)
				handle(receiver, n, f, ctx)
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
	logger := matchlog.GetLoggerFromContextWithName(ctx, "")

	if i, err := r.Intersect(ctx, n, ids); err != nil {
		log.Printf("intersect failed (%d): %v", len(i), err)
	} else {
		// write out to common-ids.txt
		log.Printf("intersected %d IDs, writing out to %s", len(i), *out)
		if f, err := os.Create(*out); err == nil {
			defer f.Close()
			for _, id := range i {
				// and write it
				if _, err := f.Write(append(id, "\n"...)); err != nil {
					logger.Error(err, "failed to write intersected ID to file")
					os.Exit(1)
				}
			}
		} else {
			logger.Error(err, "failed to perform PSI")
			os.Exit(1)
		}
	}
}
