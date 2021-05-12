package dhpsi

import (
	"context"
)

func sel(ctx context.Context, f func() error) error {
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

func run(f1 func() error, f2 func() error) chan error {
	var d = make(chan error, 2)
	go func() {
		d <- f1()
	}()
	go func() {
		d <- f2()
	}()

	return d
}
