package dresources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// createTimeSkew is how much earlier than our first create request a found resource's create time
// may be, to allow for clock skew, and still count as created by that request.
const createTimeSkew = time.Minute

// createRecovery describes a create request that failed, for a resource whose name the bundle
// chooses and the API keeps unique. T is the SDK type the create and the lookup return.
type createRecovery[T any] struct {
	// name identifies the resource in messages, e.g. the endpoint name or the dashboard path.
	name string
	// start is when the first create request was sent.
	start time.Time
	// get looks the resource up by name. It must fail with a 404 when the resource is missing.
	get func(ctx context.Context) (*T, error)
	// createTime returns when the found resource was created, or false if the response omits it.
	createTime func(found *T) (time.Time, bool)
	// create repeats the create request.
	create func(ctx context.Context) (*T, error)
}

// recoverCreate handles a create request that failed with createErr. A slow create is answered
// with 504 DEADLINE_EXCEEDED, which does not say whether the resource was created, so repeating
// the request blindly can fail with "already exists" for a resource that the next deploy would
// then also fail to create. Instead, look the resource up: use it if our request created it, and
// repeat the request while it is missing. A repeat that fails because the resource exists means an
// earlier request has landed in the meantime, so the next look finds it.
//
// Errors that are not transient, and a done context, are returned as is.
func recoverCreate[T any](ctx context.Context, rc createRecovery[T], createErr error) (*T, error) {
	if ctx.Err() != nil || !retry.IsTransient(ctx, createErr) {
		return nil, createErr
	}

	lastErr := createErr
	for attempt := 1; ; attempt++ {
		found, err := retry.With(ctx, retry.IsTransientRead, func() (*T, error) {
			return rc.get(ctx)
		})
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err == nil {
			// Without the time check, a resource that existed before the deploy would be taken over:
			// our request would have failed with "already exists" only if it reached that check.
			created, ok := rc.createTime(found)
			if !ok {
				return nil, fmt.Errorf("%w; %s exists but has no create time, so it is not known to be created by this deployment", createErr, rc.name)
			}
			if created.Before(rc.start.Add(-createTimeSkew)) {
				return nil, fmt.Errorf("%w; %s exists but was not created by this deployment", createErr, rc.name)
			}
			// Info, not Warn: the deploy got the resource it asked for, so there is nothing to act on.
			log.Infof(ctx, "using %s, created despite the failed create request: %s", rc.name, createErr)
			return found, nil
		}
		if !apierr.IsMissing(err) {
			log.Warnf(ctx, "checking whether %s was created: %s", rc.name, err)
			return nil, createErr
		}
		if attempt > retry.MaxRetries {
			return nil, lastErr
		}
		if err := retry.Pause(ctx, lastErr); err != nil {
			return nil, err
		}
		created, err := rc.create(ctx)
		if err == nil {
			return created, nil
		}
		if !retry.IsTransient(ctx, err) && !isAlreadyExists(err) {
			return nil, err
		}
		lastErr = err
	}
}

// isAlreadyExists reports whether a create request failed because the resource exists. Features,
// dashboards and vector search endpoints answer with 409 ALREADY_EXISTS, serving endpoints with
// 400 RESOURCE_ALREADY_EXISTS (observed on AWS, 2026-11).
func isAlreadyExists(err error) bool {
	return errors.Is(err, apierr.ErrAlreadyExists) || errors.Is(err, apierr.ErrResourceAlreadyExists)
}
