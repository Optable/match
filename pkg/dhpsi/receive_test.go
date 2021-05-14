package dhpsi

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"testing"

	"github.com/optable/match/test/emails"
)

const (
	ReceiverTestBodyLen = 3000
	ReceiverTestLen     = ReceiverTestBodyLen + TestCommonLen
)

// test receiver and return the addr string
func r_receiverInit(common []byte, intersectionsBus chan<- []byte, errs chan<- error) (addr string, err error) {
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
			go r_receiverHandle(common, conn, intersectionsBus, errs)
		}
	}()
	return ln.Addr().String(), nil
}

func r_receiverHandle(common []byte, conn net.Conn, intersectionsBus chan<- []byte, errs chan<- error) {
	defer close(intersectionsBus)
	r := initTestDataSource(common, ReceiverTestBodyLen)
	rec := NewReceiver(conn)
	ii, err := rec.Intersect(context.Background(), int64(ReceiverTestLen), r)
	for _, intersection := range ii {
		intersectionsBus <- intersection
	}
	if err != nil {
		// hmm - send this to the main thread with a channel
		errs <- err
	}
}

func TestReceiver(t *testing.T) {
	// generate common data
	common := emails.Common(TestCommonLen)
	// setup channels
	var intersectionsBus = make(chan []byte)
	var errs = make(chan error, 2)
	addr, err := r_receiverInit(common, intersectionsBus, errs)
	if err != nil {
		t.Fatal(err)
	}

	// send
	go func() {
		r := initTestDataSource(common, SenderTestBodyLen)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			errs <- fmt.Errorf("sender: %v", err)
		}
		s := NewSender(conn)
		err = s.Send(context.Background(), int64(SenderTestLen), r)
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
		t.Fatal(err)
	default:
	}
	// right amount?
	if len(common)/emails.HashLen != len(intersections) {
		t.Errorf("expected %d intersections and got %d", len(common)/emails.HashLen, len(intersections))
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
			t.Fatalf("expected to intersect, got %s != %s (%d %d)", s1, s2, len(s1), len(s2))
		}
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
