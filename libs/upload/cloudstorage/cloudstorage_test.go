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
