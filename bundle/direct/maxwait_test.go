package direct

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/deployplan"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceMaxWait(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		set     bool
		want    time.Duration
		wantErr string
	}{
		{name: "unset", want: maxWaitUnset},
		{name: "seconds", value: "90", set: true, want: 90 * time.Second},
		{name: "zero does not wait", value: "0", set: true, want: 0},
		{name: "typo", value: "6O", set: true, wantErr: `invalid DATABRICKS_BUNDLE_RESOURCE_MAX_WAIT="6O"`},
		{name: "negative", value: "-5", set: true, wantErr: `invalid DATABRICKS_BUNDLE_RESOURCE_MAX_WAIT="-5"`},
		{name: "duration syntax is not accepted", value: "1m", set: true, wantErr: `invalid DATABRICKS_BUNDLE_RESOURCE_MAX_WAIT="1m"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			if tc.set {
				ctx = env.Set(ctx, bundleenv.ResourceMaxWaitVariable, tc.value)
			}

			got, err := resourceMaxWait(ctx)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUnitMaxWait(t *testing.T) {
	const configured = 30 * time.Second

	tests := []struct {
		name       string
		maxWait    time.Duration
		action     deployplan.ActionType
		dependents int
		want       time.Duration
	}{
		{name: "create without dependents is capped", maxWait: configured, action: deployplan.Create, want: configured},
		{name: "create with dependents keeps full wait", maxWait: configured, action: deployplan.Create, dependents: 1, want: maxWaitUnset},
		{name: "recreate with dependents keeps full wait", maxWait: configured, action: deployplan.Recreate, dependents: 1, want: maxWaitUnset},
		{name: "delete ignores dependents", maxWait: configured, action: deployplan.Delete, dependents: 2, want: configured},
		{name: "unset stays unset", maxWait: maxWaitUnset, action: deployplan.Create, want: maxWaitUnset},
		{name: "unset stays unset for delete", maxWait: maxWaitUnset, action: deployplan.Delete, dependents: 2, want: maxWaitUnset},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, unitMaxWait(tc.maxWait, tc.action, tc.dependents))
		})
	}
}

// TestWaitCappedAbandonsPoll covers the shape produced by retries.Poll, which every resource
// wait goes through: the parent deadline wins over the resource's own timeout and surfaces as
// ErrTimedOut rather than a context error.
func TestWaitCappedAbandonsPoll(t *testing.T) {
	polls := 0
	_, err := waitCapped(t.Context(), 50*time.Millisecond, "test resource", func(ctx context.Context) (any, error) {
		return retries.Poll(ctx, time.Hour, func() (*struct{}, *retries.Err) {
			polls++
			return nil, retries.Continues("still provisioning")
		})
	})

	require.NoError(t, err)
	assert.Positive(t, polls)
}

// TestWaitCappedAbandonsBareContextError covers the other shape: retryWith returns ctx.Err()
// directly when the cap lands while it sleeps between transient-error retries.
func TestWaitCappedAbandonsBareContextError(t *testing.T) {
	_, err := waitCapped(t.Context(), 20*time.Millisecond, "test resource", waitForCtx)
	require.NoError(t, err)
}

func TestWaitCappedPropagatesFailure(t *testing.T) {
	sentinel := errors.New("index failed to provision")

	_, err := waitCapped(t.Context(), time.Minute, "test resource", func(ctx context.Context) (any, error) {
		return retries.Poll(ctx, time.Hour, func() (*struct{}, *retries.Err) {
			return nil, retries.Halt(sentinel)
		})
	})

	require.ErrorIs(t, err, sentinel)
}

// TestWaitCappedPropagatesCancellation asserts the cap does not swallow the deployment being
// cancelled or timing out as a whole, which would carry on past the user's interrupt.
func TestWaitCappedPropagatesCancellation(t *testing.T) {
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := waitCapped(ctx, time.Minute, "test resource", waitForCtx)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("parent deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()

		// Cap is longer than the parent deadline, so the parent is what expires.
		_, err := waitCapped(ctx, time.Minute, "test resource", waitForCtx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestWaitCappedUnsetAddsNoDeadline(t *testing.T) {
	_, err := waitCapped(t.Context(), maxWaitUnset, "test resource", func(ctx context.Context) (struct{}, error) {
		_, ok := ctx.Deadline()
		assert.False(t, ok, "unset cap must not impose a deadline")
		return struct{}{}, nil
	})
	require.NoError(t, err)
}

func TestWaitCappedZeroDoesNotWait(t *testing.T) {
	_, err := waitCapped(t.Context(), 0, "test resource", waitForCtx)
	require.NoError(t, err)
}

// waitForCtx blocks until the context is done and reports its error, standing in for a wait
// that never observes its resource becoming ready.
func waitForCtx(ctx context.Context) (struct{}, error) {
	<-ctx.Done()
	return struct{}{}, ctx.Err()
}
