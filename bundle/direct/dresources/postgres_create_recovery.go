package dresources

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
)

// postgresCreateTimeSkew is how much earlier than our first create request a found resource's
// create_time may be, to allow for clock skew, and still count as created by that request.
// A variable so that tests can change it.
var postgresCreateTimeSkew = time.Minute

// postgresCreateRecovery describes a Lakebase resource whose create request failed with a
// transient error. T is the SDK type of the resource and R the remote type DoCreate returns.
type postgresCreateRecovery[T, R any] struct {
	// name is the full resource name, e.g. projects/p/branches/b.
	name string
	// get reads the resource.
	get func(ctx context.Context) (*T, error)
	// create repeats the create request. On success it returns a function that waits for the
	// operation the request started.
	create func(ctx context.Context) (func() (string, R, error), error)
	// use returns the resource the lookup found.
	use func(ctx context.Context, found *T) (string, R, error)
}

// recoverPostgresCreate handles a Lakebase create request that failed with a transient error. A
// slow create is answered with 504 DEADLINE_EXCEEDED ("The create request timed out and may still
// be completing"), so the request may create the resource anyway. Look for the resource: use it if
// it is found, and repeat the request while it is missing. A repeated request that fails because
// the resource already exists means an earlier one has landed since, so the next look finds it.
func recoverPostgresCreate[T, R any](ctx context.Context, rc postgresCreateRecovery[T, R], createErr error) (string, R, error) {
	var zero R
	lastErr := createErr
	for attempt := 1; ; attempt++ {
		found, err := retry.With(ctx, retry.IsTransientRead, func() (*T, error) {
			return rc.get(ctx)
		})
		if ctx.Err() != nil {
			return "", zero, ctx.Err()
		}
		if err == nil {
			return rc.use(ctx, found)
		}
		if !apierr.IsMissing(err) {
			log.Warnf(ctx, "checking whether %s was created: %s", rc.name, err)
			return "", zero, createErr
		}
		if attempt > retry.MaxRetries {
			return "", zero, lastErr
		}
		if err := retry.Pause(ctx, lastErr); err != nil {
			return "", zero, err
		}
		wait, err := rc.create(ctx)
		if err == nil {
			return wait()
		}
		if !retry.IsTransient(ctx, err) && !isPostgresAlreadyExists(err) {
			return "", zero, err
		}
		lastErr = err
	}
}

// isPostgresAlreadyExists reports whether a Lakebase create request failed because the resource
// exists. Branches answer with 409 ALREADY_EXISTS; projects answer with 400 BAD_REQUEST and
// "A project with this resource name already exists in the workspace."
func isPostgresAlreadyExists(err error) bool {
	if errors.Is(err, apierr.ErrAlreadyExists) {
		return true
	}
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.StatusCode == http.StatusBadRequest && strings.Contains(apiErr.Message, "already exists")
}

// createdByRequestAt reports whether a resource with createTime can have been created by a request
// first sent at start. Without replace_existing, a resource that existed before the request would
// have made it fail with "already exists". That holds only if the request reached the backend's
// check, so look at when the resource was created rather than adopt one the bundle did not create.
func createdByRequestAt(createTime *sdktime.Time, start time.Time) bool {
	return createTime != nil && !createTime.AsTime().Before(start.Add(-postgresCreateTimeSkew))
}
