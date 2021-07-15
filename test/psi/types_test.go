// black box testing of all PSIs
package psi_test

import (
	"context"
	"fmt"
	"io"

	"github.com/optable/match/pkg/dhpsi"
	"github.com/optable/match/pkg/npsi"
)

type test_size struct {
	scenario                          string
	commonLen, senderLen, receiverLen int
}

var test_sizes = []test_size{
	{"emptySenderSize", 0, 0, 1000},
	{"emptyReceiverSize", 0, 1000, 0},
	{"sameSize", 100, 100, 100},
	{"smallSize", 100, 10000, 1000},
	{"mediumSize", 1000, 100000, 10000},
}

const (
	psiDHPSI = iota
	psiNPSI
)

type sender interface {
	Send(ctx context.Context, n int64, identifiers <-chan []byte) error
}

type receiver interface {
	Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error)
}

func newSender(protocol int, rw io.ReadWriter) (sender, error) {
	switch protocol {
	case psiDHPSI:
		return dhpsi.NewSender(rw), nil
	case psiNPSI:
		return npsi.NewSender(rw), nil

	default:
		return nil, fmt.Errorf("PSI sender protocol %d not supported", protocol)
	}
}

func newReceiver(protocol int, rw io.ReadWriter) (receiver, error) {
	switch protocol {
	case psiDHPSI:
		return dhpsi.NewReceiver(rw), nil
	case psiNPSI:
		return npsi.NewReceiver(rw), nil

	default:
		return nil, fmt.Errorf("PSI receiver protocol %d not supported", protocol)
	}
}
