// black box testing of all PSIs
package psi_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"testing"

	"github.com/optable/match/test/emails"
)

// test receiver and return the addr string
func r_receiverInit(protocol int, common []byte, commonLen, receiverLen int, intersectionsBus chan<- []byte, errs chan<- error) (addr string, err error) {
	ln, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return "", err
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				// handle error
			}
			go r_receiverHandle(protocol, common, commonLen, receiverLen, conn, intersectionsBus, errs)
		}
	}()
	return ln.Addr().String(), nil
}

func r_receiverHandle(protocol int, common []byte, commonLen, receiverLen int, conn net.Conn, intersectionsBus chan<- []byte, errs chan<- error) {
	defer close(intersectionsBus)
	r := initTestDataSource(common, receiverLen-commonLen)

	rec, _ := newReceiver(protocol, conn)
	ii, err := rec.Intersect(context.Background(), int64(receiverLen), r)
	for _, intersection := range ii {
		intersectionsBus <- intersection
	}
	if err != nil {
		// hmm - send this to the main thread with a channel
		errs <- err
	}
}

// take the common chunk from the emails generator
// and turn it into prefixed sha512 hashes
func parseCommon(b []byte) (out []string) {
	for i := 0; i < len(b)/emails.HashLen; i++ {
		// make one
		one := make([]byte, len(emails.Prefix)+hex.EncodedLen(len(b[i*emails.HashLen:i*emails.HashLen+emails.HashLen])))
		// copy the prefix first and then the
		// hex string
		copy(one, emails.Prefix)
		hex.Encode(one[len(emails.Prefix):], b[i*emails.HashLen:i*emails.HashLen+emails.HashLen])
		out = append(out, string(one))
	}
	return
}

func testReceiver(protocol int, common []byte, s test_size) error {
	// setup channels
	var intersectionsBus = make(chan []byte)
	var errs = make(chan error, 2)
	addr, err := r_receiverInit(protocol, common, s.commonLen, s.receiverLen, intersectionsBus, errs)
	if err != nil {
		return err
	}

	// send operation
	go func() {
		r := initTestDataSource(common, s.senderLen-s.commonLen)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			errs <- fmt.Errorf("sender: %v", err)
		}
		snd, _ := newSender(protocol, conn)
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
	// right amount?
	if len(common)/emails.HashLen != len(intersections) {
		return fmt.Errorf("expected %d intersections and got %d", len(common)/emails.HashLen, len(intersections))
	}
	// sort intersections
	sort.Slice(intersections, func(i, j int) bool {
		return string(intersections[i]) > string(intersections[j])
	})
	// sort common
	c := parseCommon(common)
	sort.Slice(c, func(i, j int) bool {
		return string(c[i]) > string(c[j])
	})

	// matching?
	for k, v := range intersections {
		s1 := string(c[k])
		s2 := string(v)
		if s1 != s2 {
			return fmt.Errorf("expected to intersect, got %s != %s (%d %d)", s1, s2, len(s1), len(s2))
		}
	}
	return nil
}

func TestDHPSIReceiver(t *testing.T) {
	for _, s := range test_sizes {
		t.Logf("testing scenario %s", s.scenario)
		// generate common data
		common := emails.Common(s.commonLen)
		// test
		if err := testReceiver(psiDHPSI, common, s); err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}
	}
}

func TestNPSIReceiver(t *testing.T) {
	for _, s := range test_sizes {
		t.Logf("testing scenario %s", s.scenario)
		// generate common data
		common := emails.Common(s.commonLen)
		// test
		if err := testReceiver(psiNPSI, common, s); err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}
	}
}
