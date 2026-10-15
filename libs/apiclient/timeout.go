package apiclient

import (
	"context"
	"io"
	"sync"
	"time"
)

// Ported from github.com/databricks/databricks-sdk-go/httpclient/timeout_context.go.
//
// The HTTP timeout is an inactivity timeout, as in the previous SDK: it is
// extended every time the request or response body is read, so large uploads
// and downloads are not cut off by a fixed deadline.

const defaultHTTPTimeout = 60 * time.Second

type timeoutTicker struct {
	cancel   context.CancelFunc
	done     <-chan struct{}
	timeout  time.Duration
	lock     sync.Mutex
	deadline time.Time
}

func newTimeoutContext(ctx context.Context, timeout time.Duration) (context.Context, *timeoutTicker) {
	ctx, cancel := context.WithCancel(ctx)
	t := &timeoutTicker{
		cancel:   cancel,
		done:     ctx.Done(),
		timeout:  timeout,
		deadline: time.Now().Add(timeout),
	}
	go t.run()
	return ctx, t
}

// Tick extends the deadline to now plus the timeout.
func (t *timeoutTicker) Tick() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.deadline = time.Now().Add(t.timeout)
}

func (t *timeoutTicker) Cancel() {
	t.cancel()
}

func (t *timeoutTicker) currentDeadline() time.Time {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.deadline
}

func (t *timeoutTicker) run() {
	for {
		ttl := time.Until(t.currentDeadline())
		if ttl <= 0 {
			t.cancel()
			return
		}
		timer := time.NewTimer(ttl)
		select {
		case <-timer.C:
		case <-t.done:
			timer.Stop()
			return
		}
	}
}

// tickingReadCloser extends the timeout on every read and, if cancelOnClose
// is set, releases the timeout context when closed.
type tickingReadCloser struct {
	rc            io.ReadCloser
	t             *timeoutTicker
	cancelOnClose bool
}

func (r tickingReadCloser) Read(p []byte) (int, error) {
	defer r.t.Tick()
	return r.rc.Read(p)
}

func (r tickingReadCloser) Close() error {
	if r.cancelOnClose {
		defer r.t.Cancel()
	}
	return r.rc.Close()
}
