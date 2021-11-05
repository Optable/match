// black box testing of all PSIs
package psi_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/optable/match/pkg/psi"
	"github.com/optable/match/test/emails"
)

// will output len(common)+bodyLen identifiers
func initTestDataSource(common []byte, bodyLen, hashLen int) <-chan []byte {
	return emails.Mix(common, bodyLen, hashLen)
}

// test receiver and return the addr string
func s_receiverInit(protocol int, common []byte, commonLen, receiverLen, hashLen int, errs chan<- error) (addr string, err error) {
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
			go s_receiverHandle(protocol, common, commonLen, receiverLen, hashLen, conn, errs)
		}
	}()
	return ln.Addr().String(), nil
}

func s_receiverHandle(protocol int, common []byte, commonLen, receiverLen, hashLen int, conn net.Conn, errs chan<- error) {
	r := initTestDataSource(common, receiverLen-commonLen, hashLen)
	// do a nil receive, ignore the results
	rec, _ := psi.NewReceiver(psi.Protocol(protocol), conn)
	_, err := rec.Intersect(context.Background(), int64(receiverLen), r)
	if err != nil {
		errs <- err
	}
}

func testSender(protocol int, addr string, common []byte, commonLen, senderLen, hashLen int) error {
	// test sender
	r := initTestDataSource(common, senderLen-commonLen, hashLen)
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
	var errs = make(chan error, 2)
	defer close(errs)
	for _, s := range test_sizes {
		t.Logf("testing scenario %s", s.scenario)
		// generate common data
		common := emails.Common(s.commonLen, s.hashLen)
		// init a test receiver server
		addr, err := s_receiverInit(p, common, s.commonLen, s.receiverLen, s.hashLen, errs)
		if err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}

		// errors?
		select {
		case err := <-errs:
			t.Fatalf("%s: %v", s.scenario, err)
		default:
		}

		// test sender
		err = testSender(p, addr, common, s.commonLen, s.senderLen, 32)
		if err != nil {
			t.Fatalf("%s: %v", s.scenario, err)
		}
	}

	for _, hashLen := range hashLenSizes {
		hashLenTest := test_size{"same size with hash length", 100, 100, 200, hashLen}
		scenario := hashLenTest.scenario + " with hash length: " + fmt.Sprint(hashLen)
		t.Logf("testing scenario %s", scenario)
		// generate common data
		common := emails.Common(hashLenTest.commonLen, hashLen)
		// init a test receiver server
		addr, err := s_receiverInit(p, common, hashLenTest.commonLen, hashLenTest.receiverLen, hashLenTest.hashLen, errs)
		if err != nil {
			t.Fatalf("%s: %v", scenario, err)
		}

		// errors?
		select {
		case err := <-errs:
			t.Fatalf("%s: %v", scenario, err)
		default:
		}

		// test sender
		err = testSender(p, addr, common, hashLenTest.commonLen, hashLenTest.senderLen, hashLenTest.hashLen)
		if err != nil {
			t.Fatalf("%s: %v", hashLenTest.scenario, err)
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

func TestKKRTPSISender(t *testing.T) {
	testSenderByProtocol(psi.KKRTPSI, t)
}
