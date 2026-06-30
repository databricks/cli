// Package cloudstorage issues unauthenticated, idempotent HTTP requests to cloud
// object storage (S3, Azure Blob, GCS) using short-lived presigned URLs. The
// URLs are self-authenticating, so the client deliberately carries no Databricks
// credentials. It owns the inactivity-stall and exact-length guards and a capped
// retry-with-backoff policy; callers supply the bytes via a [Body] and interpret
// the returned [Response].
package cloudstorage

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/databricks/sdk-go/core/ops"
)

// defaultIdleTimeout cuts a transfer that stops making progress: if the request
// body is not read for this long, no bytes are moving and the attempt is cancelled.
var defaultIdleTimeout = 60 * time.Second

var (
	errStalled   = errors.New("transfer stalled (no progress within the idle timeout)")
	errShortRead = errors.New("body produced fewer bytes than expected (was the source modified mid-transfer?)")
)

// Response is the captured result of a presigned request. Unlike a Databricks
// API call, cloud requests expose the raw status and headers (ETag, Range, 308)
// and are never mapped to a Databricks API error.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// Client issues requests to cloud object storage over presigned URLs. It holds
// only an *http.Client: no Databricks credentials, no config.
type Client struct {
	http        *http.Client
	retrier     func() ops.Retrier
	idleTimeout time.Duration
}

// New returns a Client that sends over the given HTTP client. The caller owns
// the client's transport, timeouts, and connection-pool sizing; it must not
// attach Databricks credentials, and should not set a whole-request
// http.Client.Timeout (it would abort a legitimately long transfer; bound the
// operation with the request context instead).
func New(httpClient *http.Client) *Client {
	return &Client{
		http:        httpClient,
		retrier:     newRetrier,
		idleTimeout: defaultIdleTimeout,
	}
}

// Attempt performs a single request and returns the raw response. body may be
// nil for a bodyless request (a status query or an abort).
func (c *Client) Attempt(ctx context.Context, method, url string, headers map[string]string, body Body) (*Response, error) {
	var rdr io.Reader
	var bodyLen int64
	if body != nil {
		var err error
		if rdr, err = body.Reader(); err != nil {
			return nil, err
		}
		bodyLen = body.Size()
	}

	// Cancel the attempt on two conditions, each with a distinct cause: the
	// transfer stalls (a dead connection that accepts no more bytes but never
	// errors), or the source comes up short (truncated mid-transfer). Both are
	// told apart from the caller cancelling ctx; the stall is retried, the short
	// read is not.
	attemptCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	req, err := http.NewRequestWithContext(attemptCtx, method, url, rdr)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if bodyLen > 0 && req.Body != nil {
		// A non-bytes source (a file section) is not length-detected by NewRequest,
		// so set Content-Length explicitly. Enforce the exact length and guard
		// against a stall; the body is read as bytes move to the socket, so a gap
		// longer than IdleTimeout means the transfer has stalled. The response phase
		// is bounded by the transport's ResponseHeaderTimeout.
		req.ContentLength = bodyLen
		req.Body = stallGuardedBody(exactBody(req.Body, bodyLen, cancel), c.idleTimeout, cancel)
	}

	// Trace connection reuse so a slow request can be attributed to connection
	// behavior (e.g. HTTP/2 multiplexing onto one connection vs a fresh one).
	var connReused, connWasIdle bool
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			connReused, connWasIdle = info.Reused, info.WasIdle
		},
	}))

	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		// Surface our own cancellations as distinct errors; a caller ctx
		// cancellation stays as-is. errShortRead carries its detail message.
		if ctx.Err() == nil {
			switch cause := context.Cause(attemptCtx); {
			case errors.Is(cause, errShortRead):
				err = cause
			case errors.Is(cause, errStalled):
				err = errStalled
			}
		}
		log.Debugf(ctx, "request failed: method=%s host=%s req_bytes=%d duration_ms=%d error=%v",
			method, req.URL.Host, bodyLen, time.Since(start).Milliseconds(), err)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	duration := time.Since(start)
	if err != nil {
		return nil, err
	}
	log.Debugf(ctx, "response: method=%s host=%s status=%d proto=%s conn_reused=%t conn_was_idle=%t req_bytes=%d duration_ms=%d",
		method, req.URL.Host, resp.StatusCode, resp.Proto, connReused, connWasIdle, bodyLen, duration.Milliseconds())
	return &Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: respBody}, nil
}

// Send performs Attempt with capped retries: transient network errors, the
// retryable statuses in retryStatusCodes (honoring Retry-After), and stalled
// attempts are retried with exponential backoff. A fresh request and body reader
// are built per attempt, so no explicit rewind is needed. Exhausting retries on
// a retryable status returns that status as an error.
func (c *Client) Send(ctx context.Context, method, url string, headers map[string]string, body Body) (*Response, error) {
	var result *Response
	op := func(ctx context.Context) error {
		resp, err := c.Attempt(ctx, method, url, headers, body)
		if err != nil {
			return err // network error or inactivity cut; the retrier decides
		}
		// Surface a failure status as an APIError and let the retrier decide: it
		// retries the retryable statuses (honoring Retry-After) and transient
		// transport failures, and returns everything else. A non-retryable status
		// (403 URL expired, 412 already exists, ...) therefore reaches the caller as
		// this error, which carries the status, headers, and body for inspection.
		// FromHTTPError returns nil for 2xx and 3xx, so both pass through as a
		// successful response (the http.Client already follows real redirects).
		if apiErr := apierr.FromHTTPError(resp.StatusCode, resp.Header, resp.Body); apiErr != nil {
			return apiErr
		}
		result = resp
		return nil
	}
	if err := ops.Execute(ctx, op, ops.WithRetrier(c.retrier)); err != nil {
		return nil, err
	}
	return result, nil
}

// IsURLExpired reports whether err (as returned by Send) is one of the known
// cloud-provider "presigned URL expired" errors: a 403 with the AWS or Azure
// signature-expiry body.
func IsURLExpired(err error) bool {
	aerr, ok := errors.AsType[*apierr.APIError](err)
	if !ok || aerr.HTTPStatusCode() != http.StatusForbidden {
		return false
	}
	var e struct {
		XMLName    xml.Name `xml:"Error"`
		Code       string   `xml:"Code"`
		Message    string   `xml:"Message"`
		AuthDetail string   `xml:"AuthenticationErrorDetail"`
	}
	if uerr := xml.Unmarshal(aerr.HTTPBody(), &e); uerr != nil {
		return false
	}
	switch e.Code {
	case "AuthenticationFailed": // Azure
		return strings.Contains(e.AuthDetail, "Signature not valid in the specified time frame")
	case "AccessDenied": // AWS
		return e.Message == "Request has expired"
	default:
		return false
	}
}

// stallReader wraps a request body so a transfer that stops making progress is
// cancelled. The transport reads the body as it writes bytes to the socket, so a
// gap longer than idle between reads means no bytes are moving -- a stalled or
// dead connection. A transfer making any progress keeps resetting the timer and
// so cannot be cut, unlike a whole-attempt or throughput-floor deadline.
type stallReader struct {
	r      io.Reader
	idle   time.Duration
	cancel context.CancelCauseFunc
	timer  *time.Timer
}

func newStallReader(r io.Reader, idle time.Duration, cancel context.CancelCauseFunc) *stallReader {
	return &stallReader{r: r, idle: idle, cancel: cancel}
}

func (s *stallReader) Read(p []byte) (int, error) {
	// Arm on the first read (after the connection is established, so dial time is
	// not charged against the idle budget) and reset on each later read.
	if s.timer == nil {
		s.timer = time.AfterFunc(s.idle, func() { s.cancel(errStalled) })
	} else {
		s.timer.Reset(s.idle)
	}
	n, err := s.r.Read(p)
	if err != nil {
		// EOF (transfer done) or a read error: stop arming so the response phase is
		// not subject to the stall timer.
		s.timer.Stop()
	}
	return n, err
}

// stallGuardedBody wraps a request body's reads with stall detection while
// preserving its Close.
func stallGuardedBody(body io.ReadCloser, idle time.Duration, cancel context.CancelCauseFunc) io.ReadCloser {
	return &stallGuardedReadCloser{Closer: body, sr: newStallReader(body, idle, cancel)}
}

type stallGuardedReadCloser struct {
	io.Closer
	sr *stallReader
}

func (b *stallGuardedReadCloser) Read(p []byte) (int, error) {
	return b.sr.Read(p)
}

// exactReader enforces that a body delivers exactly the declared length. If the
// source ends early -- e.g. the file was truncated during the transfer -- it
// cancels the attempt with errShortRead rather than letting a short body be sent
// (silent corruption) or surfacing net/http's opaque ContentLength-mismatch
// error (which would then be pointlessly retried).
type exactReader struct {
	r      io.Reader
	left   int64
	cancel context.CancelCauseFunc
}

func (e *exactReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	e.left -= int64(n)
	if err == io.EOF && e.left > 0 {
		cause := fmt.Errorf("%w: stream ended %d bytes short of the expected length", errShortRead, e.left)
		e.cancel(cause)
		return n, cause
	}
	return n, err
}

// exactBody wraps a request body's reads with the exact-length check while
// preserving its Close.
func exactBody(body io.ReadCloser, n int64, cancel context.CancelCauseFunc) io.ReadCloser {
	return &exactReadCloser{Closer: body, er: &exactReader{r: body, left: n, cancel: cancel}}
}

type exactReadCloser struct {
	io.Closer
	er *exactReader
}

func (b *exactReadCloser) Read(p []byte) (int, error) {
	return b.er.Read(p)
}
