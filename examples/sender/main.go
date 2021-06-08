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

	"github.com/optable/match/pkg/dhpsi"
)

const (
	defaultAddress        = "127.0.0.1:6667"
	defaultSenderFileName = "sender-ids.txt"
)

func usage() {
	log.Printf("Usage: sender [-a address] [-in file]\n")
	flag.PrintDefaults()
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

func main() {
	var addr = flag.String("a", defaultAddress, "The receiver address")
	var file = flag.String("in", defaultSenderFileName, "A list of IDs terminated with a newline")
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

	// count lines
	n, err := count(f)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("operating on %s with %d IDs", *file, n)

	// rewind
	f.Seek(0, io.SeekStart)

	c, err := net.Dial("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	s := dhpsi.NewSender(c)
	err = s.SendFromReader(context.Background(), n, f)
	if err != nil {
		log.Fatal(err)
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
