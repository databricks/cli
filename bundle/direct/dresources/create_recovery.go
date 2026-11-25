package dresources

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// createTimeSkew is how much earlier than our first create request a found resource's create time
// may be, to allow for clock skew, and still count as created by that request.
const createTimeSkew = time.Minute

// createRecovery describes a create request that failed, for a resource whose name the bundle
// chooses and the API keeps unique. T is the SDK type the lookup returns and R the result of
// DoCreate's create step; for most resources R is *T (see useFound and repeatCreate).
type createRecovery[T, R any] struct {
	// name identifies the resource in messages, e.g. the endpoint name or the dashboard path.
	name string
	// start is when the first create request was sent.
	start time.Time
	// get looks the resource up by name. It must fail with a 404 when the resource is missing.
	get func(ctx context.Context) (*T, error)
	// createTime returns when the found resource was created, or false if the response omits it.
	createTime func(found *T) (time.Time, bool)
	// use turns a found resource that this deployment created into the result, e.g. by waiting for
	// it to become ready.
	use func(ctx context.Context, found *T) (R, error)
	// alreadyExists reports whether a repeated create request failed because the name is taken:
	// isAlreadyExists, or isPostgresAlreadyExists for the Lakebase resources that answer with a 400.
	alreadyExists func(err error) bool
	// createdByCaller, if set, reports whether the found resource was created by the deploying
	// user. Within the create-time window, it tells our resource apart from one another user
	// created under the same name.
	createdByCaller func(ctx context.Context, found *T) (bool, error)
	// create repeats the create request. On success it returns finish, which completes the create,
	// e.g. by waiting for the operation the request started. An error from finish is not a failed
	// create request, so it is returned as is.
	create func(ctx context.Context) (finish func() (R, error), err error)
}

// useFound is createRecovery.use for a create whose result is the resource the lookup returns.
func useFound[T any](_ context.Context, found *T) (*T, error) {
	return found, nil
}

// repeatCreate is createRecovery.create for a create request that returns the resource itself.
func repeatCreate[T any](create func(ctx context.Context) (*T, error)) func(ctx context.Context) (func() (*T, error), error) {
	return func(ctx context.Context) (func() (*T, error), error) {
		created, err := create(ctx)
		if err != nil {
			return nil, err
		}
		return func() (*T, error) { return created, nil }, nil
	}
}

// recoverCreate handles a create request that failed with createErr. A slow create is answered
// with 504 DEADLINE_EXCEEDED, which does not say whether the resource was created, so repeating
// the request blindly can fail with "already exists" for a resource that the next deploy would
// then also fail to create. Instead, look the resource up: use it if our request created it, and
// repeat the request while it is missing. A repeat that fails because the resource exists means an
// earlier request has landed in the meantime, so the next look finds it.
//
// Errors that leave no doubt about the outcome, and a done context, are returned as is.
//
// This is the one implementation of this recovery; TestCreateRecoveryCoverage checks that every
// resource either uses it, marks an idempotent create retrySafe, or is listed in
// createRecoveryOptOut.
func recoverCreate[T, R any](ctx context.Context, rc createRecovery[T, R], createErr error) (R, error) {
	var zero R
	if ctx.Err() != nil || !isUncertainCreateError(ctx, createErr) {
		return zero, createErr
	}

	lastErr := createErr
	for attempt := 1; ; attempt++ {
		found, err := retry.With(ctx, isUncertainRead, func() (*T, error) {
			return rc.get(ctx)
		})
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if err == nil {
			// Without the time check, a resource that existed before the deploy would be taken over:
			// our request would have failed with "already exists" only if it reached that check.
			created, ok := rc.createTime(found)
			if !ok {
				return zero, &recoveryError{msg: rc.name + " exists but has no create time, so it is not known to be created by this deployment", err: createErr}
			}
			if created.Before(rc.start.Add(-createTimeSkew)) {
				return zero, &recoveryError{msg: rc.name + " exists but was not created by this deployment", err: createErr}
			}
			if rc.createdByCaller != nil {
				ours, err := rc.createdByCaller(ctx, found)
				if err != nil {
					log.Warnf(ctx, "checking who created %s: %s", rc.name, err)
					return zero, createErr
				}
				if !ours {
					return zero, &recoveryError{msg: rc.name + " exists but was created by another user", err: createErr}
				}
			}
			result, err := rc.use(ctx, found)
			if err != nil {
				return zero, err
			}
			// Info, not Warn: the deploy got the resource it asked for, so there is nothing to act on.
			log.Infof(ctx, "using %s, created despite the failed create request: %s", rc.name, strings.TrimSpace(diag.FormatAPIErrorSummary(createErr)))
			return result, nil
		}
		if !apierr.IsMissing(err) {
			log.Warnf(ctx, "checking whether %s was created: %s", rc.name, err)
			return zero, createErr
		}
		if attempt > retry.MaxRetries {
			return zero, lastErr
		}
		if err := retry.Pause(ctx, lastErr); err != nil {
			return zero, err
		}
		finish, err := rc.create(ctx)
		if err == nil {
			return finish()
		}
		if !isUncertainCreateError(ctx, err) && !rc.alreadyExists(err) {
			return zero, err
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

// sdkTime converts a create time from an SDK response for createRecovery.createTime.
func sdkTime(t *sdktime.Time) (time.Time, bool) {
	if t == nil {
		return time.Time{}, false
	}
	return t.AsTime(), true
}

// unixMilliTime converts a create time in Unix milliseconds for createRecovery.createTime. Zero
// means the response omits it.
func unixMilliTime(ms int64) (time.Time, bool) {
	return time.UnixMilli(ms), ms != 0
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
// dashboards, vector search endpoints, AI Gateway services (checked for model provider services)
// and Lakebase branches answer with 409 ALREADY_EXISTS; serving endpoints, Unity Catalog volumes and
// registered models with 400 RESOURCE_ALREADY_EXISTS; database catalogs and Unity Catalog catalogs
// with 400 CATALOG_ALREADY_EXISTS; Unity Catalog schemas with 400 SCHEMA_ALREADY_EXISTS (observed
// on AWS, 2026-11).
func isAlreadyExists(err error) bool {
	if errors.Is(err, apierr.ErrAlreadyExists) || errors.Is(err, apierr.ErrResourceAlreadyExists) {
		return true
	}
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && (apiErr.ErrorCode == "CATALOG_ALREADY_EXISTS" || apiErr.ErrorCode == "SCHEMA_ALREADY_EXISTS")
}

// createdByCurrentUser is createRecovery.createdByCaller for resources whose read response names
// the principal that created them, as Unity Catalog's created_by does: a user name, or the
// application id of a service principal.
func createdByCurrentUser(ctx context.Context, client *databricks.WorkspaceClient, createdBy string) (bool, error) {
	me, err := client.CurrentUser.Me(ctx, iam.MeRequest{Attributes: "", ExcludedAttributes: "", ForceSendFields: nil})
	if err != nil {
		return false, err
	}
	return createdBy == me.UserName, nil
}

// isPostgresAlreadyExists is isAlreadyExists for Lakebase resources. Besides 409 ALREADY_EXISTS,
// projects, databases and roles answer a create of an existing name with 400 BAD_REQUEST, e.g. "A
// project with this resource name already exists in the workspace." The API has no error code for
// it, so match the message.
func isPostgresAlreadyExists(err error) bool {
	if isAlreadyExists(err) {
		return true
	}
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.StatusCode == http.StatusBadRequest && strings.Contains(apiErr.Message, "already exists")
}
