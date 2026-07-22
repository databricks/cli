package cloudstorage

import (
	"context"
	"errors"
	"net/http"
	"syscall"
	"testing"

	"github.com/databricks/sdk-go/core/apierr"
)

func TestIsRetriable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"stall", errStalled, true},
		{"short read", errShortRead, false},
		{"canceled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
		{"transient network", syscall.ECONNRESET, true},
		{"retryable status", apierr.FromHTTPError(http.StatusServiceUnavailable, nil, nil), true},
		{"non-retryable status", apierr.FromHTTPError(http.StatusForbidden, nil, nil), false},
		{"generic", errors.New("connection reset"), false},
	}
	for _, tc := range cases {
		if got := IsRetriable(tc.err); got != tc.want {
			t.Errorf("%s: IsRetriable = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsRetryableStatus(t *testing.T) {
	if !IsRetryableStatus(http.StatusServiceUnavailable) {
		t.Error("503 should be retryable")
	}
	if IsRetryableStatus(http.StatusNotFound) {
		t.Error("404 should not be retryable")
	}
}
