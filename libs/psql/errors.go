package psql

import (
	"errors"
	"strings"
)

// errRetryable is a sentinel error used to mark retryable errors.
var errRetryable = errors.New("retryable")

// retryableError wraps an error to indicate it can be retried.
type retryableError struct {
	err error
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) Unwrap() error {
	return e.err
}

func (e *retryableError) Is(target error) bool {
	return target == errRetryable
}

// nonRetryablePatterns contains error message patterns that indicate
// permanent failures that should not be retried.
var nonRetryablePatterns = []string{
	"does not exist",        // role "..." does not exist, database "..." does not exist
	"authentication failed", // password authentication failed for user "..."
}

// isNonRetryableError checks if the error message indicates a permanent failure.
func isNonRetryableError(stderr string) bool {
	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(stderr, pattern) {
			return true
		}
	}
	return false
}
