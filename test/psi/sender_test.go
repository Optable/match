// black box testing of all PSIs
package psi_test

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/optable/match/pkg/psi"
	"github.com/optable/match/test/emails"
)

// will output len(common)+bodyLen identifiers
func initTestDataSource(common []byte, bodyLen int) <-chan []byte {
	return emails.Mix(common, bodyLen)
}

// test receiver and return the addr string
func s_receiverInit(protocol int, common []byte, commonLen, receiverLen int) (addr string, err error) {
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
			go s_receiverHandle(protocol, common, commonLen, receiverLen, conn)
		}
	}()
	return ln.Addr().String(), nil
}

func s_receiverHandle(protocol int, common []byte, commonLen, receiverLen int, conn net.Conn) {
	r := initTestDataSource(common, receiverLen-commonLen)
	// do a nil receive, ignore the results
	rec, _ := psi.NewReceiver(psi.Protocol(protocol), conn)
	_, err := rec.Intersect(context.Background(), int64(receiverLen), r)
	if err != nil {
		// hmm - send this to the main thread with a channel
		log.Print(err)
	}
}

func testSender(protocol int, addr string, common []byte, commonLen, senderLen int) error {
	// test sender
	r := initTestDataSource(common, senderLen-commonLen)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	snd, _ := psi.NewSender(psi.Protocol(protocol), conn)
	err = snd.Send(context.Background(), int64(senderLen), r)
	if err != nil {
		return err
	}
	return nil
}

func testSenderByProtocol(p int, t *testing.T) {
	for _, s := range test_sizes {
		t.Logf("testing scenario %s", s.scenario)
		// generate common data
		common := emails.Common(s.commonLen)
		// init a test receiver server
		addr, err := s_receiverInit(p, common, s.commonLen, s.receiverLen)
		if err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}
		// test sender
		err = testSender(p, addr, common, s.commonLen, s.senderLen)
		if err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}
	}
}

func TestDHPSISender(t *testing.T) {
	testSenderByProtocol(psi.DHPSI, t)
}

func TestNPSISender(t *testing.T) {
	testSenderByProtocol(psi.NPSI, t)
}

func TestBPSISender(t *testing.T) {
	testSenderByProtocol(psi.BPSI, t)
}
