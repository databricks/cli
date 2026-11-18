package dresources

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/databricks/cli/libs/diag"
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
// Errors that leave no doubt about the outcome, and a done context, are returned as is.
func recoverCreate[T any](ctx context.Context, rc createRecovery[T], createErr error) (*T, error) {
	if ctx.Err() != nil || !isUncertainCreateError(ctx, createErr) {
		return nil, createErr
	}

	lastErr := createErr
	for attempt := 1; ; attempt++ {
		found, err := retry.With(ctx, isUncertainRead, func() (*T, error) {
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
				return nil, &recoveryError{msg: rc.name + " exists but has no create time, so it is not known to be created by this deployment", err: createErr}
			}
			if created.Before(rc.start.Add(-createTimeSkew)) {
				return nil, &recoveryError{msg: rc.name + " exists but was not created by this deployment", err: createErr}
			}
			// Info, not Warn: the deploy got the resource it asked for, so there is nothing to act on.
			log.Infof(ctx, "using %s, created despite the failed create request: %s", rc.name, strings.TrimSpace(diag.FormatAPIErrorSummary(createErr)))
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
		if !isUncertainCreateError(ctx, err) && !isAlreadyExists(err) {
			return nil, err
		}
		lastErr = err
	}
}

// isUncertainCreateError reports whether a create request failed in a way that leaves it unknown
// whether the resource was created. Besides the transient errors retry.IsTransient covers, these are:
//   - 499 CANCELLED: the backend cancelled its own handling of the request (observed on model
//     provider service creates, 2026-11), possibly after the resource was written;
//   - an HTTP/2 stream reset, e.g. "response body: stream error: stream ID N; INTERNAL_ERROR"
//     (observed on AI Gateway creates, 2026-11). It may come before or after the request was
//     handled.
//
// Recovery looks the resource up before repeating the request, so treating these as uncertain
// cannot create a duplicate.
func isUncertainCreateError(ctx context.Context, err error) bool {
	if retry.IsTransient(ctx, err) {
		return true
	}
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		// Match the status, not apierr.ErrCancelled: a mapped error_code takes precedence over it.
		return apiErr.StatusCode == statusCancelled
	}
	// net/http bundles its http2 types unexported, so match the message of http2StreamError.
	return strings.Contains(err.Error(), "stream error: stream ID ")
}

// statusCancelled is the non-standard status the API uses for CANCELLED.
const statusCancelled = 499

// isUncertainRead reports whether the lookup after a failed create is worth repeating.
func isUncertainRead(ctx context.Context, err error) bool {
	return retry.IsTransientRead(ctx, err) || isUncertainCreateError(ctx, err)
}

// recoveryError says why a resource found after a failed create request is not used, and wraps
// that request's error so the bundle error output still shows its status code and details. The
// 504s these requests get carry an empty message, so the message is left out when it is empty.
type recoveryError struct {
	msg string
	err error
}

func (e *recoveryError) Error() string {
	if m := e.err.Error(); m != "" {
		return e.msg + ": " + m
	}
	return e.msg
}

func (e *recoveryError) Unwrap() error { return e.err }

// isAlreadyExists reports whether a create request failed because the resource exists. Features,
// dashboards, vector search endpoints and AI Gateway services (checked for model provider services)
// answer with 409 ALREADY_EXISTS, serving endpoints with
// 400 RESOURCE_ALREADY_EXISTS (observed on AWS, 2026-11).
func isAlreadyExists(err error) bool {
	return errors.Is(err, apierr.ErrAlreadyExists) || errors.Is(err, apierr.ErrResourceAlreadyExists)
}
