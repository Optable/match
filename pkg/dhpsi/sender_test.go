package dhpsi

import (
	"context"
	"io"
	"log"
	"net"
	"testing"

	"github.com/optable/match/test/emails"
)

var (
	TestCardinalityCommon = 101
	TestCardinalityEmails = 1000
	TestCardinality       = TestCardinalityEmails - TestCardinalityCommon
)

func initTestDataSource(common []byte) io.ReadCloser {
	// get an io pipe to read results
	i, o := io.Pipe()
	go func() {
		matchables := emails.Mix(common, TestCardinality)
		for matchable := range matchables {
			if _, err := o.Write(matchable); err != nil {
				return
			}
		}
	}()
	return i
}

// test receiver and return the addr string
func s_receiverInit(common []byte) (addr string, err error) {
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
			go s_receiverHandle(common, conn)
		}
	}()
	return ln.Addr().String(), nil
}

func s_receiverHandle(common []byte, conn net.Conn) {
	r := initTestDataSource(common)
	rec := NewReceiver(conn)
	_, err := rec.Intersect(context.Background(), int64(TestCardinalityEmails), r)
	if err != nil {
		// hmm - send this to the main thread with a channel
		log.Print(err)
	}
}

func TestSender(t *testing.T) {
	// generate common data
	common := emails.Common(TestCardinalityCommon)
	addr, err := s_receiverInit(common)
	if err != nil {
		t.Fatal(err)
	}

	// test sender
	r := initTestDataSource(common)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSender(conn)
	err = s.Send(context.Background(), int64(TestCardinalityEmails), r)
	if err != nil {
		t.Error(err)
	}

}
