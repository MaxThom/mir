package external

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

func BackOffRetry(ctx context.Context, l zerolog.Logger, maxTime time.Duration, retryFn func() error) error {
	ctx, cancel := context.WithTimeout(ctx, maxTime)
	defer cancel()

	delay := 333 * time.Millisecond
	maxDelay := 10 * time.Second

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("connection cancelled or timed out: %w", ctx.Err())
		default:
		}

		// db, err := ConnectToDb(url, namespace, database, user, password)
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
