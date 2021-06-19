package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"runtime/pprof"

	"github.com/optable/match/internal/util"
	"github.com/optable/match/pkg/dhpsi"
	"github.com/optable/match/pkg/npsi"
	"github.com/optable/match/pkg/psi"
)

const (
	defaultProtocol       = "npsi"
	defaultAddress        = "127.0.0.1:6667"
	defaultSenderFileName = "sender-ids.txt"
)

func usage() {
	log.Printf("Usage: sender [-proto protocol] [-a address] [-in file]\n")
	flag.PrintDefaults()
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

func main() {
	var protocol = flag.String("proto", defaultProtocol, "the psi protocol (dhpsi,npsi)")
	var addr = flag.String("a", defaultAddress, "The receiver address")
	var file = flag.String("in", defaultSenderFileName, "A list of IDs terminated with a newline")
	var showHelp = flag.Bool("h", false, "Show help message")
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
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

	// count lines
	log.Printf("counting lines in %s", *file)
	n, err := util.Count(f)
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
	s := newSender(*protocol, c)
	ids := util.Exhaust(n, f)
	err = s.Send(context.Background(), n, ids)
	if err != nil {
		log.Fatal(err)
	}
}

func newSender(protocol string, rw io.ReadWriter) psi.Sender {
	switch protocol {
	case "npsi":
		return npsi.NewSender(rw)
	case "dhpsi":
		return dhpsi.NewSender(rw)
	}

	return nil
}
