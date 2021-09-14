package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"

	"github.com/optable/match/internal/util"
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
	var protocol = flag.String("proto", defaultProtocol, "the psi protocol (bpsi,npsi,dhpsi,kkrt)")
	var addr = flag.String("a", defaultAddress, "The receiver address")
	var file = flag.String("in", defaultSenderFileName, "A list of IDs terminated with a newline")
	var showHelp = flag.Bool("h", false, "Show help message")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
	}

	// validate protocol
	var psi_type psi.Protocol
	switch *protocol {
	case "bpsi":
		psi_type = psi.BPSI
	case "npsi":
		psi_type = psi.NPSI
	case "dhpsi":
		psi_type = psi.DHPSI
	case "kkrt":
		psi_type = psi.KKRTPSI
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
	// enable nagle
	switch v := c.(type) {
	case *net.TCPConn:
		v.SetNoDelay(false)
	}
	s, _ := psi.NewSender(psi_type, c)
	ids := util.Exhaust(n, f)
	err = s.Send(context.Background(), n, ids)
	if err != nil {
		log.Fatal(err)
	}
}
