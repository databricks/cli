package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTransient(t *testing.T) {
	ctx := t.Context()
	assert.True(t, IsTransient(ctx, &apierr.APIError{StatusCode: http.StatusGatewayTimeout, Message: "TEMPORARILY_UNAVAILABLE"}))
	assert.True(t, IsTransient(ctx, &apierr.APIError{StatusCode: http.StatusInternalServerError}))
	assert.True(t, IsTransient(ctx, &apierr.APIError{StatusCode: http.StatusRequestTimeout}))
	// Retried by the SDK already.
	assert.False(t, IsTransient(ctx, &apierr.APIError{StatusCode: http.StatusServiceUnavailable}))
	assert.False(t, IsTransient(ctx, &apierr.APIError{StatusCode: http.StatusGatewayTimeout, Message: "deadline exceeded"}))
	assert.False(t, IsTransient(ctx, &apierr.APIError{StatusCode: http.StatusForbidden}))
	assert.False(t, IsTransient(ctx, errors.New("boom")))
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
