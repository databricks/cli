package call

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/sdk-go/core/ops"
	"github.com/databricks/cli/libs/tmp/options/internaloptions"
)

func TestOptionsApply_AllFields(t *testing.T) {
	provider := func() ops.Retrier { return nil }
	limiter := stubLimiter{}

	opts := []Option{
		WithRetrier(provider),
		WithLimiter(limiter),
		WithTimeout(3 * time.Second),
	}

	cfg := internaloptions.CallOptions{}
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	if cfg.Retrier == nil {
		t.Error("expected Retrier provider")
	}
	if cfg.RateLimiter != ops.Limiter(limiter) {
		t.Error("RateLimiter mismatch")
	}
	if cfg.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v", cfg.Timeout)
	}
}

func TestWithDisableRetry_OverridesEarlierRetrier(t *testing.T) {
	cfg := internaloptions.CallOptions{}
	if err := WithRetrier(func() ops.Retrier { return nil })(&cfg); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if cfg.Retrier == nil {
		t.Fatal("expected Retrier to be set after WithRetrier")
	}
	if err := WithDisableRetry()(&cfg); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if cfg.Retrier != nil {
		t.Fatal("expected Retrier to be cleared after WithDisableRetry")
	}
}

type stubLimiter struct{}

func (stubLimiter) Wait(_ context.Context) error { return nil }
