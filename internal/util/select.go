package util

import (
	"context"
)

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
