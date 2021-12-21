// black box testing of all PSIs
package psi_test

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"testing"

	"github.com/optable/match/pkg/psi"
	"github.com/optable/match/test/emails"
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

// test receiver and return the addr string
func r_receiverInit(protocol psi.Protocol, common []byte, commonLen, receiverLen, hashLen int, intersectionsBus chan<- []byte, errs chan<- error) (addr string, err error) {
	cert, err := tls.X509KeyPair(exampleCert, exampleKey)
	if err != nil {
		return "", err
	}
	conf := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:", conf)
	if err != nil {
		return "", err
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				// handle error
				errs <- err
			}
			go r_receiverHandle(protocol, common, commonLen, receiverLen, hashLen, conn, intersectionsBus, errs)
		}
	}()
	return ln.Addr().String(), nil
}

func r_receiverHandle(protocol psi.Protocol, common []byte, commonLen, receiverLen, hashLen int, conn net.Conn, intersectionsBus chan<- []byte, errs chan<- error) {
	defer close(intersectionsBus)
	r := initTestDataSource(common, receiverLen-commonLen, hashLen)

	rec, _ := psi.NewReceiver(protocol, conn)
	ii, err := rec.Intersect(context.Background(), int64(receiverLen), r)
	for _, intersection := range ii {
		intersectionsBus <- intersection
	}
	if err != nil {
		errs <- err
	}
}

// take the common chunk from the emails generator
// and turn it into prefixed sha512 hashes
func parseCommon(b []byte, hashLen int) (out []string) {
	for i := 0; i < len(b)/hashLen; i++ {
		// make one
		one := make([]byte, len(emails.Prefix)+hex.EncodedLen(len(b[i*hashLen:i*hashLen+hashLen])))
		// copy the prefix first and then the
		// hex string
		copy(one, emails.Prefix)
		hex.Encode(one[len(emails.Prefix):], b[i*hashLen:i*hashLen+hashLen])
		out = append(out, string(one))
	}
	return
}

func testReceiver(protocol psi.Protocol, common []byte, s test_size, deterministic bool) error {
	// setup channels
	var intersectionsBus = make(chan []byte)
	var errs = make(chan error, 2)
	addr, err := r_receiverInit(protocol, common, s.commonLen, s.receiverLen, s.hashLen, intersectionsBus, errs)
	if err != nil {
		return err
	}

	// send operation
	go func() {
		r := initTestDataSource(common, s.senderLen-s.commonLen, s.hashLen)
		conf := &tls.Config{
			InsecureSkipVerify: true,
		}
		conn, err := tls.Dial("tcp", addr, conf)
		if err != nil {
			errs <- fmt.Errorf("sender: %v", err)
		}
		snd, _ := psi.NewSender(protocol, conn)
		err = snd.Send(context.Background(), int64(s.senderLen), r)
		if err != nil {
			errs <- fmt.Errorf("sender: %v", err)
		}
	}()

	// intersection?
	var intersections [][]byte
	for i := range intersectionsBus {
		intersections = append(intersections, i)
	}
	// errors?
	select {
	case err := <-errs:
		return err
	default:
	}

	// turn the common chunk into a slice of
	// string IDs
	var c = parseCommon(common, s.hashLen)
	// is this a deterministic PSI? if not remove all false positives first
	if !deterministic {
		// filter out intersections to
		// have only IDs present in common
		intersections = filterIntersect(intersections, c)
	}

	// right amount?
	if len(common)/s.hashLen != len(intersections) {
		return fmt.Errorf("expected %d intersections and got %d", len(common)/s.hashLen, len(intersections))
	}
	// sort intersections
	sort.Slice(intersections, func(i, j int) bool {
		return string(intersections[i]) > string(intersections[j])
	})
	// sort common
	sort.Slice(c, func(i, j int) bool {
		return string(c[i]) > string(c[j])
	})

	// matching?
	for k, v := range intersections {
		s1 := c[k]
		s2 := string(v)
		if s1 != s2 {
			return fmt.Errorf("expected to intersect, got %s != %s (%d %d)", s1, s2, len(s1), len(s2))
		}
	}
	return nil
}

func filterIntersect(intersections [][]byte, common []string) [][]byte {
	var out [][]byte
	// index common
	var c = make(map[string]bool)
	for _, id := range common {
		c[id] = true
	}

	// go over intersections
	// and make sure its a member of common
	for _, id := range intersections {
		if c[string(id)] {
			out = append(out, id)
		}
	}
	return out
}

func TestDHPSIReceiver(t *testing.T) {
	for _, s := range test_sizes {
		t.Logf("testing scenario %s", s.scenario)
		// generate common data
		common := emails.Common(s.commonLen, s.hashLen)
		// test
		if err := testReceiver(psi.ProtocolDHPSI, common, s, true); err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}
	}

	for _, hashLen := range hashLenSizes {
		hashLenTest := test_size{"same size with hash digest length", 100, 100, 200, hashLen}
		scenario := hashLenTest.scenario + " with hash digest length: " + fmt.Sprint(hashDigestLen(hashLen))
		t.Logf("testing scenario %s", scenario)
		// generate common data
		common := emails.Common(hashLenTest.commonLen, hashLen)
		// test
		if err := testReceiver(psi.ProtocolDHPSI, common, hashLenTest, true); err != nil {
			t.Fatalf("%s: %v", hashLenTest.scenario, err)
		}
	}
}

func TestNPSIReceiver(t *testing.T) {
	for _, s := range test_sizes {
		t.Logf("testing scenario %s", s.scenario)
		// generate common data
		common := emails.Common(s.commonLen, s.hashLen)
		// test
		if err := testReceiver(psi.ProtocolNPSI, common, s, true); err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}
	}

	for _, hashLen := range hashLenSizes {
		hashLenTest := test_size{"same size with hash digest length", 100, 100, 200, hashLen}
		scenario := hashLenTest.scenario + " with hash digest length: " + fmt.Sprint(hashDigestLen(hashLen))
		t.Logf("testing scenario %s", scenario)
		// generate common data
		common := emails.Common(hashLenTest.commonLen, hashLen)
		// test
		if err := testReceiver(psi.ProtocolNPSI, common, hashLenTest, true); err != nil {
			t.Fatalf("%s: %v", hashLenTest.scenario, err)
		}
	}
}

func TestBPSIReceiver(t *testing.T) {
	for _, s := range test_sizes {
		t.Logf("testing scenario %s", s.scenario)
		// generate common data
		common := emails.Common(s.commonLen, s.hashLen)
		// test
		if err := testReceiver(psi.ProtocolBPSI, common, s, false); err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}
	}

	for _, hashLen := range hashLenSizes {
		hashLenTest := test_size{"same size with hash digest length", 100, 100, 200, hashLen}
		scenario := hashLenTest.scenario + " with hash digest length: " + fmt.Sprint(hashDigestLen(hashLen))
		t.Logf("testing scenario %s", scenario)
		// generate common data
		common := emails.Common(hashLenTest.commonLen, hashLen)
		// test
		if err := testReceiver(psi.ProtocolBPSI, common, hashLenTest, false); err != nil {
			t.Fatalf("%s: %v", hashLenTest.scenario, err)
		}
	}
}

func TestKKRTReceiver(t *testing.T) {
	for _, s := range test_sizes {
		t.Logf("testing scenario %s", s.scenario)
		// generate common data
		common := emails.Common(s.commonLen, s.hashLen)
		// test
		if err := testReceiver(psi.ProtocolKKRTPSI, common, s, true); err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}
	}

	for _, hashLen := range hashLenSizes {
		hashLenTest := test_size{"same size with hash length", 100, 100, 200, hashLen}
		scenario := hashLenTest.scenario + " with hash length: " + fmt.Sprint(hashDigestLen(hashLen))
		t.Logf("testing scenario %s", scenario)
		// generate common data
		common := emails.Common(hashLenTest.commonLen, hashLen)
		// test
		if err := testReceiver(psi.ProtocolKKRTPSI, common, hashLenTest, true); err != nil {
			t.Fatalf("%s: %v", hashLenTest.scenario, err)
		}
	}
}
