package psi

import (
	"context"
	"errors"
	"io"

	"github.com/optable/match/pkg/bpsi"
	"github.com/optable/match/pkg/dhpsi"
	"github.com/optable/match/pkg/kkrtpsi"
	"github.com/optable/match/pkg/npsi"
)

// Protocol is the matching protocol enumeration
type Protocol byte

const (
	ProtocolUnsupported Protocol = iota
	ProtocolDHPSI
	ProtocolNPSI
	ProtocolBPSI
	ProtocolKKRTPSI
)

var ErrUnsupportedPSIProtocol = errors.New("unsupported PSI protocol")

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
	case ProtocolUnsupported:
		fallthrough
	default:
		return nil, ErrUnsupportedPSIProtocol
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
	case ProtocolUnsupported:
		fallthrough
	default:
		return nil, ErrUnsupportedPSIProtocol
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
	case ProtocolUnsupported:
		fallthrough
	default:
		return "unsupported"
	}
}
