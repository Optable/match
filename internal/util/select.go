package util

import (
	"context"
)

// Sel runs a single stage for protocol
func Sel(ctx context.Context, f func() error) error {
	var d = make(chan error)
	go func() {
		d <- f()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-d:
		return err
	}
}

// Sels runs multiple stages for a protocol
func Sels(fs ...func() error) chan error {
	var d = make(chan error, len(fs))
	for _, f := range fs {
		f := f
		go func() {
			d <- f()
		}()
	}
	return d
}
