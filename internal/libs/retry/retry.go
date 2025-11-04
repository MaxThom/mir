package retry

import (
	"fmt"
	"strings"
	"time"
)

// RetryOnErrorContains retries the given function if the error contains the specified substring.
// It uses exponential backoff, doubling the delay after each attempt.
// Returns nil if the function succeeds, or the last error if all retries are exhausted.
//
// Parameters:
//   - fn: The function to retry
//   - errorSubstring: Only retry if the error contains this string (case-insensitive)
//   - maxAttempts: Maximum number of attempts (must be >= 1)
//   - initialDelay: Initial delay between retries (doubles after each attempt)
//
// Example:
//
//	err := retry.RetryOnErrorContains(
//	    func() error {
//	        return someOperation()
//	    },
//	    "conflict",
//	    3,
//	    100 * time.Millisecond,
//	)
func RetryOnErrorContains(
	fn func() error,
	errorSubstring string,
	maxAttempts int,
	initialDelay time.Duration,
) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	delay := initialDelay

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = fn()

		// Success - return immediately
		if lastErr == nil {
			return nil
		}

		// Check if error contains the substring
		if !strings.Contains(strings.ToLower(lastErr.Error()), strings.ToLower(errorSubstring)) {
			// Error doesn't match - don't retry
			return lastErr
		}

		// Last attempt failed - return error
		if attempt >= maxAttempts {
			return fmt.Errorf("retry exhausted after %d attempts: %w", maxAttempts, lastErr)
		}

		// Wait before retrying (exponential backoff)
		time.Sleep(delay)
		delay *= 2
	}

	return lastErr
}

// RetryOnErrorContainsWithResult is a generic version that returns a result value.
// It retries the given function if the error contains the specified substring.
//
// Parameters:
//   - fn: The function to retry that returns (T, error)
//   - errorSubstring: Only retry if the error contains this string (case-insensitive)
//   - maxAttempts: Maximum number of attempts (must be >= 1)
//   - initialDelay: Initial delay between retries (doubles after each attempt)
//
// Returns:
//   - The result value and nil error on success
//   - Zero value and error on failure
//
// Example:
//
//	devices, err := retry.RetryOnErrorContainsWithResult(
//	    func() ([]Device, error) {
//	        return db.QueryDevices()
//	    },
//	    "conflict",
//	    3,
//	    100 * time.Millisecond,
//	)
func RetryOnErrorContainsWithResult[T any](
	fn func() (T, error),
	errorSubstring string,
	maxAttempts int,
	initialDelay time.Duration,
) (T, error) {
	var zero T
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	var result T
	delay := initialDelay

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, lastErr = fn()

		// Success - return immediately
		if lastErr == nil {
			return result, nil
		}

		// Check if error contains the substring
		if !strings.Contains(strings.ToLower(lastErr.Error()), strings.ToLower(errorSubstring)) {
			// Error doesn't match - don't retry
			return zero, lastErr
		}

		// Last attempt failed - return error
		if attempt >= maxAttempts {
			return zero, fmt.Errorf("retry exhausted after %d attempts: %w", maxAttempts, lastErr)
		}

		// Wait before retrying (exponential backoff)
		time.Sleep(delay)
		delay *= 2
	}

	return zero, lastErr
}
