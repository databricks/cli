package upload

import "context"

// Limiter bounds the number of concurrent cloud-leg transfers an upload may run.
// The engine acquires one unit before each transfer (every multipart part PUT,
// the single-shot PUT, and every resumable chunk) and releases it when that
// transfer returns. Pass the same Limiter to multiple Upload calls — for example
// when copying many files at once — to cap their combined concurrency.
// Implementations must be safe for concurrent use.
type Limiter interface {
	// Acquire blocks until a slot is free or ctx is cancelled; on cancellation it
	// returns ctx.Err() and the caller must not Release.
	Acquire(ctx context.Context) error
	// Release returns a slot taken by a successful Acquire.
	Release()
}

// NewLimiter returns a Limiter permitting at most n concurrent transfers. A value
// of n <= 0 yields an unlimited limiter whose Acquire never blocks.
func NewLimiter(n int) Limiter {
	if n <= 0 {
		return unlimitedLimiter{}
	}
	return make(chanLimiter, n)
}

// unlimitedLimiter imposes no bound. It is the default when no limiter is set, so
// the transfer sites can call the limiter unconditionally.
type unlimitedLimiter struct{}

func (unlimitedLimiter) Acquire(context.Context) error { return nil }
func (unlimitedLimiter) Release()                      {}

// chanLimiter is a counting semaphore backed by a buffered channel: a send takes a
// slot, a receive returns one.
type chanLimiter chan struct{}

func (c chanLimiter) Acquire(ctx context.Context) error {
	select {
	case c <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c chanLimiter) Release() { <-c }

// WithLimiter bounds concurrent transfers via l, shared across Upload calls that
// pass the same Limiter. When unset, an upload's own parallelism governs its
// concurrency and there is no cross-upload bound.
func WithLimiter(l Limiter) UploadOption {
	return func(c *uploadConfig) { c.limiter = l }
}
