// Package poll waits for long-running operations whose progress the CLI
// reports itself. sdk-go waiters poll silently, so callers that print
// progress on every poll use this instead.
package poll

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

// Status is the outcome of one poll.
type Status[T any] struct {
	// Done reports that the operation reached its target state; Result is returned.
	Done   bool
	Result *T
	// Message describes the current state; it becomes the timeout error.
	Message string
}

// Until calls fn until it reports Done or returns an error, waiting between
// attempts with linear backoff (1s per attempt, up to 10s, plus jitter). When
// timeout expires it returns "timed out: <last message>".
func Until[T any](ctx context.Context, timeout time.Duration, fn func(context.Context) (Status[T], error)) (*T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for attempt := 1; ; attempt++ {
		status, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		if status.Done {
			return status.Result, nil
		}
		timer := time.NewTimer(backoff(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("timed out: %w", errors.New(status.Message))
		case <-timer.C:
		}
	}
}

func backoff(attempt int) time.Duration {
	wait := min(time.Duration(attempt)*time.Second, 10*time.Second)
	return wait + time.Duration(50+rand.IntN(701))*time.Millisecond
}
