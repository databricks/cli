package aircmd

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/stretchr/testify/assert"
)

func TestStoppedWaiting(t *testing.T) {
	assert.NoError(t, stoppedWaiting(t.Context(), errors.New("boom")))

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	// The SDK's endpoint poller reports Ctrl-C as its own timeout.
	assert.ErrorIs(t, stoppedWaiting(canceled, &retries.ErrTimedOut{}), errInterrupted)
	assert.ErrorIs(t, stoppedWaiting(canceled, context.Canceled), errInterrupted)

	expired, cancelExpired := context.WithTimeout(t.Context(), -time.Second)
	defer cancelExpired()
	assert.ErrorIs(t, stoppedWaiting(expired, context.DeadlineExceeded), errTimedOut)
	assert.ErrorIs(t, stoppedWaiting(t.Context(), &retries.ErrTimedOut{}), errTimedOut)
}

func TestModelErrorCode(t *testing.T) {
	for _, tt := range []struct {
		name      string
		err       error
		code      string
		retryable bool
	}{
		{"interrupted", fmt.Errorf("air serve %w", errInterrupted), "INTERRUPTED", true},
		{"timed out", fmt.Errorf("air register %w", errTimedOut), "TIMEOUT", true},
		{"invalid input", withCode(codeInvalidArgs, false, errors.New("uc_model must be a Unity Catalog model name")), "INVALID_ARGS", false},
		{"staging failed", withCode(codeStagingFailed, true, errors.New("copying failed")), "STAGING_FAILED", true},
		{"server unavailable", fmt.Errorf("x: %w", &apierr.APIError{ErrorCode: "TEMPORARILY_UNAVAILABLE", StatusCode: 503}), "TEMPORARILY_UNAVAILABLE", true},
		{"bad request", fmt.Errorf("x: %w", &apierr.APIError{ErrorCode: "INVALID_PARAMETER_VALUE", StatusCode: 400}), "INVALID_PARAMETER_VALUE", false},
		{"request timeout", &apierr.APIError{ErrorCode: "REQUEST_TIMEOUT", StatusCode: 408}, "REQUEST_TIMEOUT", true},
		{"5xx without a body", &apierr.APIError{StatusCode: 502}, "INTERNAL_ERROR", true},
		{"error body without a code", &apierr.APIError{ErrorCode: "<nil>", StatusCode: 400}, "INTERNAL_ERROR", false},
		{"lost connection", errors.New("connection reset by peer"), "INTERNAL_ERROR", true},
		{"endpoint not ready", withCode(codeEndpointNotReady, false, errors.New("endpoint isn't ready")), "ENDPOINT_NOT_READY", false},
	} {
		code, retryable := modelErrorCode(tt.err)
		assert.Equal(t, tt.code, code, tt.name)
		assert.Equal(t, tt.retryable, retryable, tt.name)
	}
}
