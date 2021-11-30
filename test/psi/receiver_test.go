// black box testing of all PSIs
package psi_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"testing"

	"github.com/optable/match/pkg/psi"
	"github.com/optable/match/test/emails"
)

// test receiver and return the addr string
func r_receiverInit(protocol int, common []byte, commonLen, receiverLen, hashLen int, intersectionsBus chan<- []byte, errs chan<- error) (addr string, err error) {
	ln, err := net.Listen("tcp", "127.0.0.1:")
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

func r_receiverHandle(protocol int, common []byte, commonLen, receiverLen, hashLen int, conn net.Conn, intersectionsBus chan<- []byte, errs chan<- error) {
	defer close(intersectionsBus)
	r := initTestDataSource(common, receiverLen-commonLen, hashLen)

	rec, _ := psi.NewReceiver(psi.Protocol(protocol), conn)
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

func testReceiver(protocol int, common []byte, s test_size, deterministic bool) error {
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
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			errs <- fmt.Errorf("sender: %v", err)
		}
		snd, _ := psi.NewSender(psi.Protocol(protocol), conn)
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
		if err := testReceiver(psi.DHPSI, common, s, true); err != nil {
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
		if err := testReceiver(psi.DHPSI, common, hashLenTest, true); err != nil {
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
		if err := testReceiver(psi.NPSI, common, s, true); err != nil {
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
		if err := testReceiver(psi.NPSI, common, hashLenTest, true); err != nil {
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
		if err := testReceiver(psi.BPSI, common, s, false); err != nil {
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
		if err := testReceiver(psi.BPSI, common, hashLenTest, false); err != nil {
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
		if err := testReceiver(psi.KKRTPSI, common, s, true); err != nil {
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
		if err := testReceiver(psi.KKRTPSI, common, hashLenTest, true); err != nil {
			t.Fatalf("%s: %v", hashLenTest.scenario, err)
		}
	}
}
