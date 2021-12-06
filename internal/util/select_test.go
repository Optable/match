package util

import (
	"context"
	"fmt"
	"testing"
	"time"
)

var (
	err1 = fmt.Errorf("this is an error")
)

func TestSel(t *testing.T) {
	// check normal operation
	f1 := func() error {
		return err1
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := Sel(ctx, f1); err != err1 {
		t.Errorf("expected %v, got %v", err1, err)
	}

	// check context canceled
	cancel()
	if err := Sel(ctx, f1); err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// check deadline exceeded
	f2 := func() error {
		time.Sleep(time.Second)
		return nil
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Second/10)
	defer cancel()
	if err := Sel(ctx, f2); err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}
