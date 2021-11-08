package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"

	"github.com/go-logr/logr"
	"github.com/optable/match/internal/util"
	matchlog "github.com/optable/match/pkg/log"
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

func exitOnErr(logger logr.Logger, err error, msg string) {
	if err != nil {
		logger.Error(err, msg)
		os.Exit(1)
	}
}

func main() {
	var protocol = flag.String("proto", defaultProtocol, "the psi protocol (dhpsi,npsi)")
	var addr = flag.String("a", defaultAddress, "The receiver address")
	var file = flag.String("in", defaultSenderFileName, "A list of IDs terminated with a newline")
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
	slog := matchlog.GetLogger(*verbose)

	// open file
	f, err := os.Open(*file)
	exitOnErr(slog, err, "failed to open file")

	// count lines
	log.Printf("counting lines in %s", *file)
	n, err := util.Count(f)
	exitOnErr(slog, err, "failed to count")
	log.Printf("operating on %s with %d IDs", *file, n)

	// rewind
	f.Seek(0, io.SeekStart)

	c, err := net.Dial("tcp", *addr)
	exitOnErr(slog, err, "failed to dial")
	defer c.Close()
	// enable nagle
	switch v := c.(type) {
	case *net.TCPConn:
		v.SetNoDelay(false)
	}
	s, _ := psi.NewSender(psiType, c)
	ids := util.Exhaust(n, f)
	err = s.Send(matchlog.ContextWithLogger(context.Background(), slog), n, ids)
	exitOnErr(slog, err, "failed to perform PSI")
}
