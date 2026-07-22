package files

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/databricks/cli/libs/tmp/files/v2/internal/cloudstorage"
)

func TestSlowAttemptGuardDeadline(t *testing.T) {
	tun := defaultTunables()
	tun.slowWarmup = 10
	tun.slowWindow = 200
	tun.slowFactor = 3
	tun.slowMinDeadline = 1 * time.Second

	g := &slowAttemptGuard{tun: tun}
	if d := g.deadline(); d != tun.slowColdDeadline {
		t.Fatalf("before warmup: deadline = %v, want cold-start %v", d, tun.slowColdDeadline)
	}

	// 90 fast parts and 10 slow ones: p95 falls in the slow tail (20s), so the
	// deadline is 3*20s = 60s, well above the fast bulk but below a real straggler.
	for range 90 {
		g.record(1 * time.Second)
	}
	for range 10 {
		g.record(20 * time.Second)
	}
	if d := g.deadline(); d != 60*time.Second {
		t.Fatalf("deadline = %v, want 60s (3 x p95 of 20s)", d)
	}

	// A fast network (tiny p95) is floored so the deadline never clips variance.
	g2 := &slowAttemptGuard{tun: tun}
	for range 20 {
		g2.record(100 * time.Millisecond)
	}
	if d := g2.deadline(); d != tun.slowMinDeadline {
		t.Fatalf("floored deadline = %v, want %v", d, tun.slowMinDeadline)
	}
}

// TestSendPartSoftDeadlineCancels verifies a wedged attempt (the server never
// responds) is cancelled at the soft deadline and reported as slow, promptly.
func TestSendPartSoftDeadlineCancels(t *testing.T) {
	tun := defaultTunables()
	tun.slowCheckInterval = 5 * time.Millisecond

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done(): // the client cancelled at the soft deadline
		case <-time.After(3 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)

	e := &engine{c: &Client{}, tun: tun}
	uc := &uploadContext{tun: tun, cloud: cloudstorage.New(srv.Client()), limiter: NewLimiter(0)}

	start := time.Now()
	_, slow, err := e.sendPart(t.Context(), uc, srv.URL, nil, cloudstorage.BytesBody([]byte("payload")),
		func() time.Duration { return 50 * time.Millisecond })
	elapsed := time.Since(start)

	if !slow {
		t.Fatalf("slow = false, want true (err=%v, elapsed=%v)", err, elapsed)
	}
	if elapsed > time.Second {
		t.Fatalf("sendPart returned after %v, want prompt cancel near the 50ms deadline", elapsed)
	}
}

// TestSendPartFastNotSlow verifies a fast attempt under a generous deadline is
// not flagged slow and returns the response.
func TestSendPartFastNotSlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "etag-1")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	tun := defaultTunables()
	e := &engine{c: &Client{}, tun: tun}
	uc := &uploadContext{tun: tun, cloud: cloudstorage.New(srv.Client()), limiter: NewLimiter(0)}

	resp, slow, err := e.sendPart(t.Context(), uc, srv.URL, nil, cloudstorage.BytesBody([]byte("payload")),
		func() time.Duration { return 5 * time.Second })
	if err != nil {
		t.Fatalf("sendPart: %v", err)
	}
	if slow {
		t.Fatal("slow = true, want false for a fast attempt")
	}
	if resp.Header.Get("ETag") != "etag-1" {
		t.Errorf("ETag = %q, want etag-1", resp.Header.Get("ETag"))
	}
}
