package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/optable/match/internal/util"
	"github.com/optable/match/pkg/dhpsi"
	"github.com/optable/match/pkg/npsi"
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

var out *string

func main() {
	var wg sync.WaitGroup
	var protocol = flag.String("proto", defaultProtocol, "the psi protocol (dhpsi,npsi)")
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
	switch *protocol {
	case "npsi":
		fallthrough
	case "dhpsi":
		log.Printf("operating with protocol %s", *protocol)
	default:
		log.Printf("unsupported protocol %s", *protocol)
		showUsageAndExit(0)
	}

	// open file
	f, err := os.Open(*file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// count lines
	log.Printf("counting lines in %s", *file)
	n, err := util.Count(f)
	if err != nil {
		log.Fatal(err)
	}
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
			// make the receiver
			receiver := newReceiver(*protocol, c)
			// and hand it off
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer c.Close()
				handle(receiver, n, f)
			}()

			if *once {
				wg.Wait()
				return
			}
		}
	}
}

func newReceiver(protocol string, rw io.ReadWriter) psi.Receiver {
	switch protocol {
	case "npsi":
		return npsi.NewReceiver(rw)
	case "dhpsi":
		return dhpsi.NewReceiver(rw)
	}

	return nil
}

func handle(r psi.Receiver, n int64, f io.ReadCloser) {
	defer f.Close()
	ids := util.Exhaust(n, f)
	if i, err := r.Intersect(context.Background(), n, ids); err != nil {
		log.Printf("intersect failed (%d): %v", len(i), err)
	} else {
		// write out to common-ids.txt
		if f, err := os.Create(*out); err == nil {
			defer f.Close()
			for _, id := range i {
				// and write it
				if _, err := f.Write(append(id, "\n"...)); err != nil {
					log.Fatal(err)
				}
			}
			log.Printf("intersected %d IDs into %s", len(i), *out)
		} else {
			log.Fatal(err)
		}
	}
}
