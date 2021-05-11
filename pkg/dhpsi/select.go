package dhpsi

import "context"

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
