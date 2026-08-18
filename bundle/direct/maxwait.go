package direct

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/databricks/cli/bundle/deployplan"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/retries"
)

// maxWaitUnset means "no cap configured", which must stay distinguishable from an explicit
// 0 ("do not wait at all").
const maxWaitUnset = time.Duration(-1)

// resourceMaxWait returns the cap on waiting for a resource to reach its target state, or
// maxWaitUnset when the environment variable is absent. Unlike retryInterval, a malformed
// value is an error rather than a silent fallback: ignoring a typo would restore the
// multi-hour default wait that the user was trying to shorten.
func resourceMaxWait(ctx context.Context) (time.Duration, error) {
	v, ok := bundleenv.ResourceMaxWait(ctx)
	if !ok {
		return maxWaitUnset, nil
	}
	seconds, err := strconv.Atoi(v)
	if err != nil || seconds < 0 {
		return maxWaitUnset, fmt.Errorf("invalid %s=%q: expected a non-negative number of seconds", bundleenv.ResourceMaxWaitVariable, v)
	}
	return time.Duration(seconds) * time.Second, nil
}

// unitMaxWait returns the cap to apply to a single resource, given how many resources run
// after it in the deployment graph.
//
// Deletes are capped regardless of dependents. The trade-off is accepted deliberately: state
// is dropped before the wait, so a cut-short delete leaves the resource untracked while it is
// still tearing down, and the dependency deleted after it may then be rejected for still
// having a child. Recreate's internal delete-wait is excluded structurally — it is the one
// wait never routed through here, because it releases the name for the following create.
//
// Every other action is capped only when nothing depends on this resource, since a dependent
// would otherwise act on a resource that has not reached its target state.
func unitMaxWait(maxWait time.Duration, action deployplan.ActionType, dependents int) time.Duration {
	if action == deployplan.Delete || dependents == 0 {
		return maxWait
	}
	return maxWaitUnset
}

// waitCapped runs wait under maxWait. When the cap expires the wait is abandoned with a
// warning instead of failing the deployment: state is written before the wait, so the
// resource stays tracked and the next plan reconciles it. Genuine failures still propagate,
// since retries reports those without a timeout error.
func waitCapped[T any](ctx context.Context, maxWait time.Duration, description string, wait func(context.Context) (T, error)) (T, error) {
	if maxWait == maxWaitUnset {
		return wait(ctx)
	}

	waitCtx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()

	result, err := wait(waitCtx)

	// waitCtx expired but ctx did not: the cap fired rather than the whole deployment being
	// cancelled, which must keep failing so an interrupt is not swallowed.
	if err != nil && waitCtx.Err() != nil && ctx.Err() == nil && isWaitTimeout(err) {
		log.Warnf(ctx, "Stopped waiting for %s after %s (%s); it may still be in progress", description, maxWait, bundleenv.ResourceMaxWaitVariable)
		var zero T
		return zero, nil
	}

	return result, err
}

// isWaitTimeout reports whether err is a wait that ran out of time rather than a resource
// that failed. Two shapes reach here: retries.Poll reports a deadline as ErrTimedOut wrapping
// the last poll message, while retryWith returns a bare context error when the deadline lands
// while it sleeps between transient-error retries.
func isWaitTimeout(err error) bool {
	if _, ok := errors.AsType[*retries.ErrTimedOut](err); ok {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}
