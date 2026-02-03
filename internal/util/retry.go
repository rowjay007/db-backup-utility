package util

import (
	"context"
	"time"
)

// Retry executes fn with retries and backoff.
func Retry(ctx context.Context, attempts int, backoff time.Duration, fn func() error) error {
	if attempts <= 1 {
		return fn()
	}
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
