package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
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
	slog := getLogger(*verbose)

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
	err = s.Send(logr.NewContext(context.Background(), slog), n, ids)
	exitOnErr(slog, err, "failed to perform PSI")
}
