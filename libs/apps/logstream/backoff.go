package logstream

import (
	"context"
	"time"
)

const (
	initialReconnectBackoff = 200 * time.Millisecond
	maxReconnectBackoff     = 5 * time.Second
)

// backoffStrategy manages exponential backoff for reconnection attempts.
type backoffStrategy struct {
	initial time.Duration
	max     time.Duration
	current time.Duration
}

// newBackoffStrategy creates a new backoffStrategy with the given initial and max durations.
func newBackoffStrategy(initial, max time.Duration) *backoffStrategy {
	return &backoffStrategy{
		initial: initial,
		max:     max,
		current: initial,
	}
}

// Wait blocks until the current backoff duration has elapsed or the context is canceled.
func (b *backoffStrategy) Wait(ctx context.Context) error {
	if b.current <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(b.current):
		return nil
	}
}

// Next increases the backoff duration exponentially, capped at the max duration.
func (b *backoffStrategy) Next() {
	b.current = min(b.current*2, b.max)
}

// Reset returns the backoff duration to the initial value.
func (b *backoffStrategy) Reset() {
	b.current = b.initial
}
