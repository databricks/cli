package cloudstorage

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/sdk-go/core/apierr"
	"github.com/databricks/sdk-go/core/ops"
)

// fastClient returns a Client whose retrier backs off in milliseconds, so retry
// tests do not sleep.
func fastClient(srv *httptest.Server) *Client {
	fast := ops.BackoffPolicy{Initial: time.Millisecond, Maximum: time.Millisecond, Factor: 1}
	c := New(srv.Client())
	c.retrier = func() ops.Retrier {
		return &retrier{backoff: fast, maxRetries: 3}
	}
	return c
}

func TestSendRetriesRetryableStatus(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := fastClient(srv).Send(t.Context(), http.MethodPut, srv.URL, nil, BytesBody([]byte("x")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want 2 (one retry)", calls.Load())
	}
}

func TestSendExhaustsRetryableStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := fastClient(srv).Send(t.Context(), http.MethodPut, srv.URL, nil, BytesBody([]byte("x")))
	aerr, ok := errors.AsType[*apierr.APIError](err)
	if !ok || aerr.HTTPStatusCode() != http.StatusServiceUnavailable {
		t.Fatalf("err = %v, want a 503 APIError after exhausting retries", err)
	}
}

func TestSendReturnsNonRetryableStatusAsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	resp, err := New(srv.Client()).Send(t.Context(), http.MethodPut, srv.URL, nil, BytesBody([]byte("x")))
	if resp != nil {
		t.Errorf("expected nil response on error, got %v", resp)
	}
	aerr, ok := errors.AsType[*apierr.APIError](err)
	if !ok || aerr.HTTPStatusCode() != http.StatusForbidden {
		t.Fatalf("err = %v, want a 403 APIError", err)
	}
	if string(aerr.HTTPBody()) != "nope" {
		t.Errorf("HTTPBody = %q, want %q", aerr.HTTPBody(), "nope")
	}
}

func TestAttemptReturnsRawResponse(t *testing.T) {
	// Attempt never maps a status to an error: a 308 (resumable continue) and its
	// Range header come back as a Response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Range", "bytes=0-41")
		w.WriteHeader(http.StatusPermanentRedirect)
	}))
	defer srv.Close()

	resp, err := New(srv.Client()).Attempt(t.Context(), http.MethodPut, srv.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusPermanentRedirect {
		t.Errorf("status = %d, want 308", resp.StatusCode)
	}
	if got := resp.Header.Get("Range"); got != "bytes=0-41" {
		t.Errorf("Range = %q, want bytes=0-41", got)
	}
}

func TestIsURLExpired(t *testing.T) {
	azure := `<?xml version="1.0"?><Error><Code>AuthenticationFailed</Code><AuthenticationErrorDetail>Signature not valid in the specified time frame</AuthenticationErrorDetail></Error>`
	aws := `<?xml version="1.0"?><Error><Code>AccessDenied</Code><Message>Request has expired</Message></Error>`
	other := `<?xml version="1.0"?><Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error>`

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"azure expired", apierr.FromHTTPError(http.StatusForbidden, nil, []byte(azure)), true},
		{"aws expired", apierr.FromHTTPError(http.StatusForbidden, nil, []byte(aws)), true},
		{"other 403", apierr.FromHTTPError(http.StatusForbidden, nil, []byte(other)), false},
		{"404", apierr.FromHTTPError(http.StatusNotFound, nil, []byte(azure)), false},
		{"nil", nil, false},
		{"non-apierr", errors.New("boom"), false},
	}
	for _, tc := range cases {
		if got := IsURLExpired(tc.err); got != tc.want {
			t.Errorf("%s: IsURLExpired = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestStallGuardStopsTimerOnClose verifies the idle timer is disarmed when the
// transport closes the request body, not only on a terminal Read. On a
// fixed-length request the transport normally issues a trailing EOF read that
// stops the timer, but Close is the reliable signal that body transmission is
// done: if the timer stayed armed into the response phase, a slow response could
// be cancelled as errStalled even though the upload succeeded. After Close, the
// cancel cause must never be set.
func TestStallGuardStopsTimerOnClose(t *testing.T) {
	var cancelled atomic.Bool
	cancel := func(error) { cancelled.Store(true) }

	// A tiny idle window so a leaked timer would fire quickly.
	const idle = 20 * time.Millisecond
	payload := []byte("payload")
	body := stallGuardedBody(io.NopCloser(bytes.NewReader(payload)), idle, cancel)

	// Simulate a transport that stops reading at Content-Length without a trailing
	// EOF read: read exactly len(payload) bytes (each Read returns n>0, nil, which
	// arms the timer), then Close. Only Close can disarm the timer here.
	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(body, buf); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	if err := body.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Wait past the idle window; a timer left armed by Close would fire here.
	time.Sleep(3 * idle)
	if cancelled.Load() {
		t.Fatal("stall timer fired after the body was fully read and closed")
	}
}

func TestExactReaderShortRead(t *testing.T) {
	var caught error
	er := &exactReader{r: bytes.NewReader(make([]byte, 10)), left: 100, cancel: func(c error) { caught = c }}

	_, err := io.ReadAll(er)
	if !errors.Is(err, errShortRead) {
		t.Fatalf("read error = %v, want errShortRead", err)
	}
	if !errors.Is(caught, errShortRead) {
		t.Errorf("attempt cancelled with %v, want errShortRead", caught)
	}
}
