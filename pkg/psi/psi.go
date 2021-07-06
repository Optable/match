package psi

import (
	"context"
	"fmt"
	"io"

	"github.com/optable/match/pkg/dhpsi"
	"github.com/optable/match/pkg/npsi"
)

const (
	DHPSI = iota
	NPSI
)

type Sender interface {
	Send(ctx context.Context, n int64, identifiers <-chan []byte) error
}

type Receiver interface {
	Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error)
}

func NewSender(protocol int, rw io.ReadWriter) (Sender, error) {
	switch protocol {
	case DHPSI:
		return dhpsi.NewSender(rw), nil
	case NPSI:
		return npsi.NewSender(rw), nil

	default:
		return nil, fmt.Errorf("PSI sender protocol %d not supported", protocol)
	}
}

func NewReceiver(protocol int, rw io.ReadWriter) (Receiver, error) {
	switch protocol {
	case DHPSI:
		return dhpsi.NewReceiver(rw), nil
	case NPSI:
		return npsi.NewReceiver(rw), nil

	default:
		return nil, fmt.Errorf("PSI receiver protocol %d not supported", protocol)
	}
}
