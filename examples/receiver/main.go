package main

import (
	"context"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/optable/match/examples/format"
	"github.com/optable/match/internal/util"
	"github.com/optable/match/pkg/psi"
)

const (
	defaultProtocol       = "npsi"
	defaultPort           = ":6667"
	defaultSenderFileName = "receiver-ids.txt"
	defaultCommonFileName = "common-ids.txt"
)

// example certs were generated from crypto/tls/generate_cert.go with the following command:
//   go run $(go env GOROOT)/src/crypto/tls/generate_cert.go --rsa-bits 2048
// --host 127.0.0.1,::1,example.com --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
var exampleCert = []byte(`-----BEGIN CERTIFICATE-----
MIIDOjCCAiKgAwIBAgIRAPhLot3LxaigqdKikGI6PtgwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAgFw03MDAxMDEwMDAwMDBaGA8yMDg0MDEyOTE2
MDAwMFowEjEQMA4GA1UEChMHQWNtZSBDbzCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBANzxrGrkW7X+MALttWLMVhpS5QmQBYgu1Vmj/2pu4w7XUVpKolyC
ZahZBDug5p60w86kzDhLBuaRF5XypAmsCZQSB1decgkf7u3JHC2/RPyWINw/uAix
kY8G3JC8Gpz+nVonlYpYON4WSQa1ZmZ2Vz8AO/qYfBJ525Dz0zf0UTgqi3gCyqzI
/yWAhOhrxN5QbrXzLwSxG7EIejsNIp3W/PD1Cxxy1ljG3O4Po1zB9m9zh+dyaa1n
zg9ltWKWrHtikYTzYkkE4KAvYRbQhRR2mdRALs/i4vmMkp8ZVGf2DD4mBPqyh4Do
/0DrcCSiTVbKBXl4J+OeQKMZRhCMFFKBavUCAwEAAaOBiDCBhTAOBgNVHQ8BAf8E
BAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/BAUwAwEB/zAdBgNV
HQ4EFgQUZ+pstuWDZ6dcdGd/7dLl6GkXucowLgYDVR0RBCcwJYILZXhhbXBsZS5j
b22HBH8AAAGHEAAAAAAAAAAAAAAAAAAAAAEwDQYJKoZIhvcNAQELBQADggEBAHcp
MpqwqKIvcdNEFj9i7yCMPzteVfH5GZeOtO4Rglwn7TzYpJhvbZDzYPd9CkFVuIOo
h+5wUTRDiT8rES9wxQ4qHjByGqzSoJ1oaIzcEsihWzdCvFzkzSjyCDgMesqnZM/6
0nzz4ZnL8c+gf0IG18KEfJu3tvm17Jdbk1Y731mwrDoGy7MNxQVKsDT6+kMdP6FX
BLwxFHGfQfq3EqIRtm8UCOaPzchn6iXveksbdXWPSK2Yk3sLX1wzqIYRA1Ia/dOR
mOQqatUkit83UM5707DjkTxS5TJh+rsLRtKP04wtTLKUr0kjwNYPBnFhhs8bJoSW
ltYQrxMCa6uPtEyQBi0=
-----END CERTIFICATE-----`)

var exampleKey = []byte(`-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDc8axq5Fu1/jAC
7bVizFYaUuUJkAWILtVZo/9qbuMO11FaSqJcgmWoWQQ7oOaetMPOpMw4SwbmkReV
8qQJrAmUEgdXXnIJH+7tyRwtv0T8liDcP7gIsZGPBtyQvBqc/p1aJ5WKWDjeFkkG
tWZmdlc/ADv6mHwSeduQ89M39FE4Kot4AsqsyP8lgIToa8TeUG618y8EsRuxCHo7
DSKd1vzw9QscctZYxtzuD6NcwfZvc4fncmmtZ84PZbVilqx7YpGE82JJBOCgL2EW
0IUUdpnUQC7P4uL5jJKfGVRn9gw+JgT6soeA6P9A63Akok1WygV5eCfjnkCjGUYQ
jBRSgWr1AgMBAAECggEANB6+saiVCeWgpdA1jczuMt+DMDJNW8bQhYjuY8ksvv+E
LWyVyITqPkBhgz99p8q0tjaiBlWMly97BOBsWeu/hrKKEM4y0Hw7/NQIVbJdL8iq
j8poO4TH9ZmExo/ZJ1fY/r9/w1b0c0+GgpKgSWN5SV9gxsjZ2/HrHdKm7PgxgLH4
Z4dPK13JgPelz6c1prDCXhCWbmkZMnJg/w+0/ZmuIlublDMO6CK4rIy+b2Hd1vqv
74PkxIc9XpZOYb7d3li117w+htgDs/03NVztBn7BykEHtHEhfFgDdTFvJe8A6msM
7qyqOJey0KLkKZZ15f+VR3sb8NBB5LO7+SUhAJxcsQKBgQDtTuwBgh5hTr+MtWJD
HsQtN5oICBgHzYXsGRGn1H51ECKGc9KM4WDXW3dCbBAS7HHTH7kywvqkXS8mfnmu
yhNgfiECIgfS7F1mX0NKxSBBFX4auhHGqYGMBw1y0EFB5GpXJZXc4G3BXI95VgLe
m87/cbmbj4lKoBQhaahdhBbcFwKBgQDuWMiEXrYxf5GhpCZ/ET4UmvqxPJbVCe79
6hFnF8pk1sq8kIn6/aF83+9L4BCVzbYmRXDya9mw/CMimGKtVDSxtvk35XCGpfY5
M1fvV0oFq03vbsHYMr3/Sb3IstO9zvhtfUHgQbuh5uoGoSRPMdlBZIikX4mOiUCc
hgS64lSc0wKBgG3FIwAzmy/xyEMjJ/faRG6SGKr8a3k4hWlH01XpwjEOLJo6+zr1
ieE0Sv8rk2fdfW1mcDld3ain/gZ1XH4QtVPeJBCjgzD66t1O1YbBloDkmzdruItH
n0gRfxQL5xO+v73eAetw2PQnh6pdsegc9GxOw8eEZsJhN86Y3CudzSEzAoGAXEM/
84WaL1T7cb/SKxPonR9U9bDHjlYXDnFCJU8fSKOgvReSYfc2QNmKjyuAIA0Oeogc
7ap0DT+89hJY+FGFSFnU5R9KzMSHqKLIYly+ya0DMTEFloQl6iGIdp1Ku8nXfsKi
8oVfdY+mfcR5ArMAL4EUJ9TXsbZNrYlvYUxlhoMCgYEAj46cxTcJJ8LKIj5sF/+j
LpFjpum1QouSsB8CbscG+0OYS3Bs9Mfh0pGGR/hgjsg5+R4PRITfzJUEYnjtW4TL
3AKbYyx2+W62SrtSv9p8yGJgdGLQaaJd4OaVDXCzllUAII3wRtB3YSUlGMpn9MkR
rp4tfIxqa0W9QmmnenEgDj0=
-----END PRIVATE KEY-----`)

func usage() {
	log.Printf("Usage: receiver [-proto protocol] [-p port] [-in file] [-out file] [-once false]\n")
	flag.PrintDefaults()
}

var out *string

func main() {
	var wg sync.WaitGroup
	var protocol = flag.String("proto", defaultProtocol, "the psi protocol (bpsi,npsi,dhpsi,kkrt)")
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
	mlog := format.GetLogger(*verbose)

	// open file
	f, err := os.Open(*file)
	format.ExitOnErr(mlog, err, "failed to open file")
	defer f.Close()

	// count lines
	log.Printf("counting lines in %s", *file)
	t := time.Now()
	n, err := util.Count(f)
	format.ExitOnErr(mlog, err, "failed to count")
	log.Printf("that took %v", time.Since(t))
	log.Printf("operating on %s with %d IDs", *file, n)

	// get a listener
	cert, err := tls.X509KeyPair(exampleCert, exampleKey)
	if err != nil {
		format.ExitOnErr(mlog, err, "failed to load certs")
	}
	conf := &tls.Config{Certificates: []tls.Certificate{cert}}
	l, err := tls.Listen("tcp", *port, conf)
	format.ExitOnErr(mlog, err, "failed to listen on tcp port")
	log.Printf("receiver listening on %s", *port)
	//defer l.Close()
	for {
		if c, err := l.Accept(); err != nil {
			format.ExitOnErr(mlog, err, "failed to accept incoming connection")
		} else {
			log.Printf("handling sender %s", c.RemoteAddr())
			f, err := os.Open(*file)
			format.ExitOnErr(mlog, err, "failed to open file")

			// make the receiver
			receiver, err := psi.NewReceiver(psiType, c)
			format.ExitOnErr(mlog, err, "failed to create receiver")
			// and hand it off
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer c.Close()
				handle(receiver, n, f, logr.NewContext(context.Background(), mlog))
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
	logger := logr.FromContextOrDiscard(ctx)
	if i, err := r.Intersect(ctx, n, ids); err != nil {
		format.ExitOnErr(logger, err, "intersect failed")
	} else {
		// write memory usage to stderr
		format.MemUsageToStdErr(logger)
		// write out to common-ids.txt
		log.Printf("intersected %d IDs, writing out to %s", len(i), *out)
		if f, err := os.Create(*out); err == nil {
			defer f.Close()
			for _, id := range i {
				// and write it
				if _, err := f.Write(append(id, "\n"...)); err != nil {
					format.ExitOnErr(logger, err, "failed to write intersected ID to file")
				}
			}
		} else {
			format.ExitOnErr(logger, err, "failed to perform PSI")
		}
	}
}
