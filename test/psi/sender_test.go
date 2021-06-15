// black box testing of all PSIs
package psi_test

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/optable/match/test/emails"
)

const (
	TestCommonLen = 100
	TestSenderLen = 10000
)

// will output len(common)+bodyLen identifiers
func initTestDataSource(common []byte, bodyLen int) <-chan []byte {
	return emails.Mix(common, bodyLen)
}

// test receiver and return the addr string
func s_receiverInit(protocol int, common []byte, totalReceiverSize int) (addr string, err error) {
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
			go s_receiverHandle(protocol, common, totalReceiverSize, conn)
		}
	}()
	return ln.Addr().String(), nil
}

func s_receiverHandle(protocol int, common []byte, totalReceiverSize int, conn net.Conn) {
	r := initTestDataSource(common, totalReceiverSize-TestCommonLen)
	// do a nil receive, ignore the results
	rec, _ := newReceiver(protocol, conn)
	_, err := rec.Intersect(context.Background(), int64(totalReceiverSize), r)
	if err != nil {
		// hmm - send this to the main thread with a channel
		log.Print(err)
	}
}

func testSender(protocol int, addr string, common []byte, totalSenderSize int, t *testing.T) {
	// test sender
	r := initTestDataSource(common, totalSenderSize-TestCommonLen)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	snd, _ := newSender(protocol, conn)
	err = snd.Send(context.Background(), int64(totalSenderSize), r)
	if err != nil {
		t.Error(err)
	}
}

func TestDHPSISender(t *testing.T) {
	// generate common data
	common := emails.Common(TestCommonLen)
	// init a test receiver server
	addr, err := s_receiverInit(psiDHPSI, common, TestReceiverLen)
	if err != nil {
		t.Fatal(err)
	}
	// test sender
	testSender(psiDHPSI, addr, common, TestSenderLen, t)
}

func TestNPSISender(t *testing.T) {
	// generate common data
	common := emails.Common(TestCommonLen)
	// init a test receiver server
	addr, err := s_receiverInit(psiNPSI, common, TestReceiverLen)
	if err != nil {
		t.Fatal(err)
	}
	// test sender
	testSender(psiNPSI, addr, common, TestSenderLen, t)
}
