package upload

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestLimiterCapacityAndCancel(t *testing.T) {
	lim := NewLimiter(1)
	ctx := t.Context()
	if err := lim.Acquire(ctx); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	// A second acquire blocks at capacity; cancelling its context unblocks it with
	// the context error, and the caller must not Release.
	cctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- lim.Acquire(cctx) }()
	select {
	case err := <-done:
		t.Fatalf("acquire should block at capacity, returned %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("blocked acquire = %v, want context.Canceled", err)
	}

	// Releasing the first slot lets a fresh acquire succeed.
	lim.Release()
	if err := lim.Acquire(ctx); err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	lim.Release()
}

func TestUnlimitedLimiter(t *testing.T) {
	lim := NewLimiter(0) // <= 0 means unlimited
	for range 5 {
		if err := lim.Acquire(t.Context()); err != nil {
			t.Fatalf("unlimited acquire: %v", err)
		}
	}
	for range 5 {
		lim.Release()
	}
}

// countingLimiter enforces a real bound (via an inner Limiter) and records the
// peak concurrency and the acquire/release counts so tests can assert the engine
// holds exactly one slot per transfer and never exceeds the bound.
type countingLimiter struct {
	inner Limiter

	mu       sync.Mutex
	cur, max int
	acquires int
	releases int
}

func newCountingLimiter(n int) *countingLimiter {
	return &countingLimiter{inner: NewLimiter(n)}
}

func (l *countingLimiter) Acquire(ctx context.Context) error {
	if err := l.inner.Acquire(ctx); err != nil {
		return err
	}
	l.mu.Lock()
	l.cur++
	l.acquires++
	l.max = max(l.max, l.cur)
	l.mu.Unlock()
	return nil
}

func (l *countingLimiter) Release() {
	l.mu.Lock()
	l.cur--
	l.releases++
	l.mu.Unlock()
	l.inner.Release()
}

// slowParts makes every cloud part/chunk PUT take d so concurrent transfers
// overlap and the peak-concurrency assertion is meaningful.
func slowParts(f *fakeServer, d time.Duration) {
	f.partHook = func(n, attempt int) (int, string, []byte) {
		time.Sleep(d)
		return 0, "", nil
	}
}

func TestMultipartLimiterBoundsConcurrency(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	slowParts(f, 15*time.Millisecond)
	c := newTestClient(t, f)

	const limit = 2
	lim := newCountingLimiter(limit)
	in := data(8 * 1024) // 8 parts at the 1024 part size
	_, err := c.Upload(t.Context(), "/Volumes/c/s/v/m.bin", bytes.NewReader(in),
		WithParallelism(8), WithLimiter(lim))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)

	if lim.max > limit {
		t.Errorf("peak concurrency %d exceeded limit %d", lim.max, limit)
	}
	if lim.max != limit {
		t.Errorf("peak concurrency %d, want it to reach the limit %d (8 parts, 8 workers)", lim.max, limit)
	}
	if lim.acquires != 8 {
		t.Errorf("acquires = %d, want 8 (one per part)", lim.acquires)
	}
	if lim.acquires != lim.releases {
		t.Errorf("unbalanced: %d acquires, %d releases", lim.acquires, lim.releases)
	}
}

func TestSingleShotAcquiresOneSlot(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	c := newTestClient(t, f)

	lim := newCountingLimiter(4)
	in := data(500) // below the 1024 single-shot threshold
	_, err := c.Upload(t.Context(), "/Volumes/c/s/v/s.bin", bytes.NewReader(in), WithLimiter(lim))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)

	if lim.acquires != 1 || lim.releases != 1 {
		t.Errorf("single-shot: acquires=%d releases=%d, want 1/1", lim.acquires, lim.releases)
	}
	if lim.max != 1 {
		t.Errorf("single-shot peak concurrency = %d, want 1", lim.max)
	}
}

func TestResumableAcquiresPerChunk(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "resumable")
	c := newTestClient(t, f)

	lim := newCountingLimiter(4)
	in := data(4 * 1024) // several resumable chunks
	_, err := c.Upload(t.Context(), "/Volumes/c/s/v/r.bin", bytes.NewReader(in), WithLimiter(lim))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)

	if lim.acquires < 2 {
		t.Errorf("resumable acquires = %d, want >= 2 (one per chunk)", lim.acquires)
	}
	if lim.acquires != lim.releases {
		t.Errorf("unbalanced: %d acquires, %d releases", lim.acquires, lim.releases)
	}
	if lim.max != 1 {
		t.Errorf("resumable is sequential; peak concurrency = %d, want 1", lim.max)
	}
}

func TestLimiterCancelNoLeak(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	slowParts(f, 50*time.Millisecond)
	c := newTestClient(t, f)

	lim := newCountingLimiter(2)
	ctx, cancel := context.WithCancel(t.Context())
	errc := make(chan error, 1)
	go func() {
		_, err := c.Upload(ctx, "/Volumes/c/s/v/c.bin", bytes.NewReader(data(16*1024)),
			WithParallelism(8), WithLimiter(lim))
		errc <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-errc // upload returns (with a context error)

	lim.mu.Lock()
	defer lim.mu.Unlock()
	if lim.acquires != lim.releases {
		t.Errorf("cancellation leaked slots: %d acquires, %d releases", lim.acquires, lim.releases)
	}
}
