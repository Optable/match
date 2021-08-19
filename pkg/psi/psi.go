package psi

import (
	"context"
	"fmt"
	"io"

	"github.com/optable/match/pkg/bpsi"
	"github.com/optable/match/pkg/dhpsi"
	"github.com/optable/match/pkg/kkrtpsi"
	"github.com/optable/match/pkg/npsi"
)

const (
	DHPSI = iota
	NPSI
	BPSI
	KKRTPSI
)

// Protocol is the matching protocol enumeration
type Protocol int

var (
	ProtocolDHPSI   Protocol = DHPSI
	ProtocolNPSI    Protocol = NPSI
	ProtocolBPSI    Protocol = BPSI
	ProtocolKKRTPSI Protocol = KKRTPSI
)

// Sender is the sender side of the PSI operation
type Sender interface {
	Send(ctx context.Context, n int64, identifiers <-chan []byte) error
}

// Receiver side of the PSI operation
type Receiver interface {
	Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error)
}

func NewSender(protocol Protocol, rw io.ReadWriter) (Sender, error) {
	switch protocol {
	case ProtocolDHPSI:
		return dhpsi.NewSender(rw), nil
	case ProtocolNPSI:
		return npsi.NewSender(rw), nil
	case ProtocolBPSI:
		return bpsi.NewSender(rw), nil
	case ProtocolKKRTPSI:
		return kkrtpsi.NewSender(rw), nil

	default:
		return nil, fmt.Errorf("PSI sender protocol %d not supported", protocol)
	}
}

func NewReceiver(protocol Protocol, rw io.ReadWriter) (Receiver, error) {
	switch protocol {
	case ProtocolDHPSI:
		return dhpsi.NewReceiver(rw), nil
	case ProtocolNPSI:
		return npsi.NewReceiver(rw), nil
	case ProtocolBPSI:
		return bpsi.NewReceiver(rw), nil
	case ProtocolKKRTPSI:
		return kkrtpsi.NewReceiver(rw), nil

	default:
		return nil, fmt.Errorf("PSI receiver protocol %d not supported", protocol)
	}
}

func (p Protocol) String() string {
	switch p {
	case ProtocolDHPSI:
		return "dhpsi"
	case ProtocolNPSI:
		return "npsi"
	case ProtocolBPSI:
		return "bpsi"
	case ProtocolKKRTPSI:
		return "kkrtpsi"
	default:
		return "undefined"
	}
}
