package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"

	"github.com/go-logr/logr"
	"github.com/optable/match/examples/format"
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

func main() {
	var protocol = flag.String("proto", defaultProtocol, "the psi protocol (bpsi,npsi,dhpsi,kkrt)")
	var addr = flag.String("a", defaultAddress, "The receiver address")
	var file = flag.String("in", defaultSenderFileName, "A list of IDs terminated with a newline")
	var verbose = flag.Int("v", 0, "Verbosity level, default to -v 0 for info level messages, -v 1 for debug messages, and -v 2 for trace level message.")
	var showHelp = flag.Bool("h", false, "Show help message")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		format.ShowUsageAndExit(usage, 0)
	}

	// validate protocol
	var psiType psi.Protocol
	switch *protocol {
	case "bpsi":
		psiType = psi.ProtocolBPSI
	case "npsi":
		psiType = psi.ProtocolNPSI
	case "dhpsi":
		psiType = psi.ProtocolDHPSI
	case "kkrt":
		psiType = psi.ProtocolKKRTPSI
	default:
		psiType = psi.ProtocolUnsupported
	}

	log.Printf("operating with protocol %s", psiType)
	// fetch stdr logger
	slog := format.GetLogger(*verbose)

	// open file
	f, err := os.Open(*file)
	format.ExitOnErr(slog, err, "failed to open file")

	// count lines
	log.Printf("counting lines in %s", *file)
	n, err := util.Count(f)
	format.ExitOnErr(slog, err, "failed to count")
	log.Printf("operating on %s with %d IDs", *file, n)

	// rewind
	f.Seek(0, io.SeekStart)

	c, err := net.Dial("tcp", *addr)
	format.ExitOnErr(slog, err, "failed to dial")
	defer c.Close()
	// enable nagle
	switch v := c.(type) {
	case *net.TCPConn:
		v.SetNoDelay(false)
	}

	s, err := psi.NewSender(psiType, c)
	format.ExitOnErr(slog, err, "failed to create sender")
	ids := util.Exhaust(n, f)
	err = s.Send(logr.NewContext(context.Background(), slog), n, ids)
	format.ExitOnErr(slog, err, "failed to perform PSI")
	format.MemUsageToStdErr(slog)
}
