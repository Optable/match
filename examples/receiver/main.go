package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

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

func memUsageToStdErr() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m) // https://cs.opensource.google/go/go/+/go1.17.1:src/runtime/mstats.go;l=107
	log.Printf("Total memory: %v\n", m.Sys)
	log.Printf("Garbage collector calls: %v\n", m.NumGC)
}

var out *string

func main() {
	var wg sync.WaitGroup
	var protocol = flag.String("proto", defaultProtocol, "the psi protocol (bpsi,npsi,dhpsi,kkrt)")
	var port = flag.String("p", defaultPort, "The receiver port")
	var file = flag.String("in", defaultSenderFileName, "A list of IDs terminated with a newline")
	out = flag.String("out", defaultCommonFileName, "A list of IDs that intersect between the receiver and the sender")
	var once = flag.Bool("once", false, "Exit after processing one receiver")

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

	// open file
	f, err := os.Open(*file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// count lines
	log.Printf("counting lines in %s", *file)
	t := time.Now()
	n, err := util.Count(f)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("that took %v", time.Since(t))
	log.Printf("operating on %s with %d IDs", *file, n)

	// get a listener
	l, err := net.Listen("tcp", *port)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("receiver listening on %s", *port)
	for {
		if c, err := l.Accept(); err != nil {
			log.Fatal(err)
		} else {
			log.Printf("handling sender %s", c.RemoteAddr())
			f, err := os.Open(*file)
			if err != nil {
				log.Fatal(err)
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
				handle(receiver, n, f)
				log.Printf("handled sender %s", c.RemoteAddr())
			}()

			if *once {
				wg.Wait()
				return
			}
		}
	}
	memUsageToStdErr()
}

func handle(r psi.Receiver, n int64, f io.ReadCloser) {
	defer f.Close()
	ids := util.Exhaust(n, f)
	if i, err := r.Intersect(context.Background(), n, ids); err != nil {
		log.Printf("intersect failed (%d): %v", len(i), err)
	} else {
		// write out to common-ids.txt
		log.Printf("intersected %d IDs, writing out to %s", len(i), *out)
		if f, err := os.Create(*out); err == nil {
			defer f.Close()
			for _, id := range i {
				// and write it
				if _, err := f.Write(append(id, "\n"...)); err != nil {
					log.Fatal(err)
				}
			}
		} else {
			log.Fatal(err)
		}
	}
}
