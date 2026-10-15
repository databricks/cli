package apiclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/logger"
	"github.com/databricks/databricks-sdk-go/logger/httplog"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/databricks/sdk-go/core/apierr"
)

// transport makes sdk-go requests look like the ones the CLI always sent: the
// User-Agent is composed from the CLI's context (command, exec id, auth type,
// CI/CD and agent detection), the config's Headers hook is applied, and
// requests are dumped at debug level.
type transport struct {
	cfg  *config.Config
	base http.RoundTripper
}

func newTransport(cfg *config.Config) http.RoundTripper {
	return &transport{cfg: cfg, base: baseTransport(cfg)}
}

func baseTransport(cfg *config.Config) http.RoundTripper {
	if cfg.HTTPTransport != nil {
		return cfg.HTTPTransport
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.InsecureSkipVerify {
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return t
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := useragent.InContext(req.Context(), useragent.AuthKey, t.cfg.AuthType)
	for _, kv := range [][2]string{
		{useragent.CicdKey, useragent.CiCdProvider()},
		{useragent.RuntimeKey, useragent.Runtime()},
		{useragent.AgentKey, useragent.AgentProvider()},
		{useragent.MetaHarnessKey, useragent.MetaHarnessProvider()},
	} {
		if kv[1] != "" {
			ctx = useragent.InContext(ctx, kv[0], kv[1])
		}
	}
	req = req.Clone(ctx)
	req.Header.Set("User-Agent", useragent.FromContext(ctx))
	if t.cfg.Headers != nil {
		if err := t.cfg.Headers(req); err != nil {
			return nil, err
		}
	}

	return t.retry(ctx, req)
}

// retry re-sends requests rejected with 429 or 503, as the previous SDK did
// for every call. sdk-go retries nothing by default.
func (t *transport) retry(ctx context.Context, req *http.Request) (*http.Response, error) {
	timeout := time.Duration(t.cfg.RetryTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultRetryTimeout
	}
	deadline := time.Now().Add(timeout)
	maxAttempts, _ := ctx.Value(maxAttemptsKey{}).(int)
	for attempt := 0; ; attempt++ {
		resp, err := t.send(ctx, req)
		if err != nil || !isRetriable(resp) || maxAttempts > 0 && attempt+1 >= maxAttempts {
			return resp, err
		}
		wait := retryDelay(resp, attempt)
		if time.Now().Add(wait).After(deadline) || req.Body != nil && req.GetBody == nil {
			return resp, nil
		}
		_ = resp.Body.Close()
		logger.Debugf(ctx, "retrying %s %s after %s: status %d", req.Method, req.URL.Path, wait, resp.StatusCode)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req = req.Clone(ctx)
			req.Body = body
		}
	}
}

const defaultRetryTimeout = 5 * time.Minute

type maxAttemptsKey struct{}

// WithMaxAttempts caps how many times requests made with ctx are sent when
// the server answers 429 or 503, for callers where such an answer is not
// transient (e.g. a feature unavailable in a region).
func WithMaxAttempts(ctx context.Context, n int) context.Context {
	return context.WithValue(ctx, maxAttemptsKey{}, n)
}

func isRetriable(resp *http.Response) bool {
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s >= 0 {
		return time.Duration(s) * time.Second
	}
	return min(time.Duration(attempt+1)*time.Second, 10*time.Second)
}

func (t *transport) send(ctx context.Context, req *http.Request) (*http.Response, error) {
	timeout := time.Duration(t.cfg.HTTPTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	timeoutCtx, ticker := newTimeoutContext(ctx, timeout)
	req = req.WithContext(timeoutCtx)
	if req.Body != nil && req.Body != http.NoBody {
		req.Body = tickingReadCloser{rc: req.Body, t: ticker}
	}
	resp, err := t.sendLogged(ctx, req)
	if err != nil {
		ticker.Cancel()
		if ctx.Err() == nil && timeoutCtx.Err() != nil {
			return nil, fmt.Errorf("request timed out after %s of inactivity", timeout)
		}
		return nil, err
	}
	if err := normalizeErrorBody(resp); err != nil {
		ticker.Cancel()
		return nil, err
	}
	if resp.Body != nil {
		resp.Body = tickingReadCloser{rc: resp.Body, t: ticker, cancelOnClose: true}
	} else {
		ticker.Cancel()
	}
	if resp != nil && resp.StatusCode >= 400 {
		// sdk-go errors keep the response headers but not the request; record
		// it so error messages can name the failing endpoint (see RequestOf).
		resp.Header.Set(requestHeader, req.Method+" "+req.URL.String())
	}
	return resp, err
}

// requestHeader is a synthetic response header carrying the request line of
// a failed call. It is never sent to the server.
const requestHeader = "X-Databricks-Cli-Request"

// RequestOf returns the method and URL of the request that produced the API
// error in err's chain, when it was sent through this package's transport.
func RequestOf(err error) (method string, u *url.URL, ok bool) {
	apiErr, isAPIErr := errors.AsType[*apierr.APIError](err)
	if !isAPIErr {
		return "", nil, false
	}
	method, rawURL, ok := strings.Cut(apiErr.HTTPHeader().Get(requestHeader), " ")
	if !ok {
		return "", nil, false
	}
	u, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", nil, false
	}
	return method, u, true
}

func (t *transport) sendLogged(ctx context.Context, req *http.Request) (*http.Response, error) {
	if !logger.Get(ctx).Enabled(ctx, logger.LevelDebug) {
		return t.base.RoundTrip(req)
	}

	var reqBody []byte
	if req.Body != nil && req.GetBody != nil {
		if body, err := req.GetBody(); err == nil {
			reqBody, _ = io.ReadAll(body)
		}
	}
	resp, err := t.base.RoundTrip(req)
	var respBody []byte
	if resp != nil && resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
	}
	logger.Debugf(ctx, "%s", httplog.RoundTripStringer{
		Request:                  req,
		Response:                 resp,
		Err:                      err,
		RequestBody:              reqBody,
		ResponseBody:             respBody,
		DebugHeaders:             t.cfg.DebugHeaders,
		DebugTruncateBytes:       t.cfg.DebugTruncateBytes,
		DebugAuthorizationHeader: true,
	}.String())
	return resp, err
}

// HTTPClient returns an http.Client that sends requests the way sdk-go clients
// built by libs/sdkclients do. It does not authenticate requests.
func HTTPClient(cfg *config.Config) *http.Client {
	return &http.Client{Transport: newTransport(cfg)}
}
