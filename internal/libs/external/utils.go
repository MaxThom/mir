package external

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// CallWithTimeout executes a function with a timeout and returns the result or an error if it times out
func CallWithTimeout[T any](ctx context.Context, timeout time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Result channel
	type result struct {
		value T
		err   error
	}
	
	resultChan := make(chan result, 1)
	
	// Execute function in goroutine
	go func() {
		value, err := fn()
		resultChan <- result{value: value, err: err}
	}()
	
	// Wait for result or timeout
	select {
	case <-ctx.Done():
		return zero, fmt.Errorf("operation timed out after %v", timeout)
	case res := <-resultChan:
		return res.value, res.err
	}
}

func BackOffRetry(ctx context.Context, l zerolog.Logger, maxTime time.Duration, retryFn func() error) error {
	ctx, cancel := context.WithTimeout(ctx, maxTime)
	defer cancel()

	delay := 333 * time.Millisecond
	maxDelay := 10 * time.Second

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled or timed out: %w", ctx.Err())
		default:
		}

		err := retryFn()
		if err == nil {
			return nil
		}

		delay = time.Duration(float64(delay) * 1.5)
		delay = min(delay, maxDelay)

		l.Warn().Err(err).Msg("failed to connect to host, retrying in " + delay.String())

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("connection cancelled or timed out: %w", ctx.Err())
		case <-timer.C:
			// Continue to next attempt
		}
	}
}
