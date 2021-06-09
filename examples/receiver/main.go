package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/optable/match/pkg/dhpsi"
)

const (
	defaultPort           = ":6667"
	defaultSenderFileName = "receiver-ids.txt"
	defaultCommonFileName = "common-ids.txt"
)

func usage() {
	log.Printf("Usage: receiver [-p port] [-in file] [-out file] [-once false]\n")
	flag.PrintDefaults()
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

var out *string

func main() {
	var wg sync.WaitGroup
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

	// open file
	f, err := os.Open(*file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// count lines
	n, err := count(f)
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
			wg.Add(1)
			f, err := os.Open(*file)
			if err != nil {
				log.Fatal(err)
			}
			go func() {
				defer wg.Done()
				handle(c, n, f)
			}()

			if *once {
				wg.Wait()
				return
			}
		}
	}
}

func handle(c net.Conn, n int64, f io.ReadCloser) {
	defer c.Close()
	defer f.Close()
	r := dhpsi.NewReceiver(c)
	if i, err := r.IntersectFromReader(context.Background(), n, f); err != nil {
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

func count(r io.Reader) (int64, error) {
	var count int64
	const lineBreak = '\n'
	buf := make([]byte, bufio.MaxScanTokenSize)
	for {
		bufferSize, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}
		var buffPosition int
		for {
			i := bytes.IndexByte(buf[buffPosition:], lineBreak)
			if i == -1 || bufferSize == buffPosition {
				break
			}
			buffPosition += i + 1
			count++
		}
		if err == io.EOF {
			break
		}
	}
	return count, nil
}
