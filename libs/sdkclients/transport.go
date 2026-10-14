package sdkclients

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/logger"
	"github.com/databricks/databricks-sdk-go/logger/httplog"
	"github.com/databricks/databricks-sdk-go/useragent"
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
	for attempt := 0; ; attempt++ {
		resp, err := t.send(ctx, req)
		if err != nil || !isRetriable(resp) {
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
// built by this package do. It does not authenticate requests.
func HTTPClient(cfg *config.Config) *http.Client {
	return &http.Client{Transport: newTransport(cfg)}
}
