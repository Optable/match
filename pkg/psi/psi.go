package psi

import "context"

type Sender interface {
	Send(ctx context.Context, n int64, identifiers <-chan []byte) error
}

type Receiver interface {
	Intersect(ctx context.Context, n int64, identifiers <-chan []byte) ([][]byte, error)
}
