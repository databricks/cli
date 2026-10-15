package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTransient(t *testing.T) {
	ctx := t.Context()
	assert.True(t, IsTransient(ctx, apierr.FromHTTPError(http.StatusGatewayTimeout, nil, []byte(`{"message":"TEMPORARILY_UNAVAILABLE"}`))))
	assert.True(t, IsTransient(ctx, apierr.FromHTTPError(http.StatusInternalServerError, nil, nil)))
	assert.True(t, IsTransient(ctx, apierr.FromHTTPError(http.StatusRequestTimeout, nil, nil)))
	assert.True(t, IsTransient(ctx, apierr.FromHTTPError(http.StatusGatewayTimeout, nil, []byte(`{"message":"deadline exceeded"}`))))
	// Retried by the transport already.
	assert.False(t, IsTransient(ctx, apierr.FromHTTPError(http.StatusServiceUnavailable, nil, nil)))
	assert.False(t, IsTransient(ctx, apierr.FromHTTPError(http.StatusTooManyRequests, nil, nil)))
	assert.False(t, IsTransient(ctx, apierr.FromHTTPError(http.StatusForbidden, nil, nil)))
	assert.False(t, IsTransient(ctx, errors.New("boom")))
}

func TestIsTransientRead(t *testing.T) {
	ctx := t.Context()
	// Everything IsTransient covers is transient for a read too.
	assert.True(t, IsTransientRead(ctx, apierr.FromHTTPError(http.StatusInternalServerError, nil, nil)))
	assert.True(t, IsTransientRead(ctx, apierr.FromHTTPError(http.StatusRequestTimeout, nil, nil)))
	// REFUSED_STREAM is the transport failure the SDK leaves unretried, so we replay the read (issue #6472).
	refusedStream := &url.Error{Op: "Get", URL: "https://example.test/api/2.0/workspace/get-status", Err: errors.New("stream error: stream ID 1; REFUSED_STREAM; received from peer")}
	assert.True(t, IsTransientRead(ctx, refusedStream))
	// Transport errors the SDK already retries for its full RetryTimeout must not be retried again.
	connRefused := &url.Error{Op: "Get", URL: "https://example.test/api/2.0/workspace/get-status", Err: errors.New("dial tcp 10.0.0.1:443: connect: connection refused")}
	assert.False(t, IsTransientRead(ctx, connRefused))
	// A cancelled request is a *url.Error too, but not REFUSED_STREAM, so it fails fast rather than logging a retry.
	cancelled := &url.Error{Op: "Get", URL: "https://example.test/api/2.0/workspace/get-status", Err: context.Canceled}
	assert.False(t, IsTransientRead(ctx, cancelled))
	// Not transient: retried by the SDK, client errors, and plain non-url errors.
	assert.False(t, IsTransientRead(ctx, apierr.FromHTTPError(http.StatusServiceUnavailable, nil, nil)))
	assert.False(t, IsTransientRead(ctx, apierr.FromHTTPError(http.StatusNotFound, nil, nil)))
	assert.False(t, IsTransientRead(ctx, apierr.FromHTTPError(http.StatusForbidden, nil, nil)))
	assert.False(t, IsTransientRead(ctx, errors.New("boom")))
}

// testContext shortens the pause between attempts.
func testContext(t *testing.T) context.Context {
	return env.Set(t.Context(), IntervalMsVariable, "1")
}

func TestInterval(t *testing.T) {
	ctx := t.Context()
	assert.Equal(t, 15*time.Second, interval(ctx))
	assert.Equal(t, 100*time.Millisecond, interval(env.Set(ctx, IntervalMsVariable, "100")))
	assert.Equal(t, 15*time.Second, interval(env.Set(ctx, IntervalMsVariable, "abc")))
}

func TestWithRetriesUntilSuccess(t *testing.T) {
	ctx := testContext(t)
	calls := 0
	got, err := With(ctx, func(context.Context, error) bool { return true }, func() (int, error) {
		calls++
		if calls <= MaxRetries {
			return 0, errors.New("fail")
		}
		return 42, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 42, got)
	assert.Equal(t, MaxRetries+1, calls)
}

func TestWithGivesUp(t *testing.T) {
	ctx := testContext(t)
	calls := 0
	_, err := With(ctx, func(context.Context, error) bool { return true }, func() (int, error) {
		calls++
		return 0, errors.New("fail")
	})
	require.EqualError(t, err, "fail")
	assert.Equal(t, MaxRetries+1, calls)
}

func TestWithDoesNotRetryWhenCheckFails(t *testing.T) {
	ctx := testContext(t)
	calls := 0
	_, err := With(ctx, func(context.Context, error) bool { return false }, func() (int, error) {
		calls++
		return 0, errors.New("fail")
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestWithStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(env.Set(t.Context(), IntervalMsVariable, "3600000"))
	calls := 0
	_, err := With(ctx, func(context.Context, error) bool { return true }, func() (int, error) {
		calls++
		cancel()
		return 0, errors.New("fail")
	})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}

func TestRetryMessage(t *testing.T) {
	apiErr := apierr.FromHTTPError(http.StatusGatewayTimeout, nil, []byte(`{"message":"boom"}`))
	assert.Equal(t, "retrying", retryMessage(errors.New("x")))
	assert.Equal(t, "retrying after 504 Gateway Timeout", retryMessage(apiErr))
	assert.Equal(t, "retrying after 504 Gateway Timeout: failed to upload /a/b: boom", retryMessage(fmt.Errorf("failed to upload /a/b: %w", apiErr)))
}
