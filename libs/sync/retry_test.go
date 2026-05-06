package sync

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/require"
)

func apiErr(status int) error {
	return &apierr.APIError{StatusCode: status, Message: http.StatusText(status)}
}

func TestIsTransientGatewayError(t *testing.T) {
	cases := map[error]bool{
		nil:                                    false,
		apiErr(http.StatusBadGateway):          true,
		apiErr(http.StatusServiceUnavailable):  true,
		apiErr(http.StatusGatewayTimeout):      true,
		apiErr(http.StatusInternalServerError): false,
		apiErr(http.StatusTooManyRequests):     false,
		apiErr(http.StatusNotFound):            false,
		errors.New("not an api error"):         false,
	}
	for err, want := range cases {
		require.Equal(t, want, isTransientGatewayError(err), "%v", err)
	}
}

func TestRetryOnTransient(t *testing.T) {
	t.Run("succeeds after retries", func(t *testing.T) {
		var calls atomic.Int32
		err := retryOnTransient(t.Context(), 30*time.Second, "test", func() error {
			if calls.Add(1) < 3 {
				return apiErr(http.StatusBadGateway)
			}
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, int32(3), calls.Load())
	})

	t.Run("does not retry non-transient", func(t *testing.T) {
		var calls atomic.Int32
		err := retryOnTransient(t.Context(), 30*time.Second, "test", func() error {
			calls.Add(1)
			return apiErr(http.StatusNotFound)
		})
		require.Error(t, err)
		require.Equal(t, int32(1), calls.Load())
	})

	t.Run("zero timeout disables retries", func(t *testing.T) {
		var calls atomic.Int32
		err := retryOnTransient(t.Context(), 0, "test", func() error {
			calls.Add(1)
			return apiErr(http.StatusBadGateway)
		})
		require.Error(t, err)
		require.Equal(t, int32(1), calls.Load())
	})

	t.Run("times out on persistent transient error", func(t *testing.T) {
		var calls atomic.Int32
		err := retryOnTransient(t.Context(), 100*time.Millisecond, "test", func() error {
			calls.Add(1)
			return apiErr(http.StatusBadGateway)
		})
		require.Error(t, err)
		require.GreaterOrEqual(t, calls.Load(), int32(1))
	})

	t.Run("honors context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := retryOnTransient(ctx, 30*time.Second, "test", func() error {
			return apiErr(http.StatusBadGateway)
		})
		require.Error(t, err)
	})
}
