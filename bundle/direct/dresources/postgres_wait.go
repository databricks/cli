package dresources

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/api"
)

// waitCreateOperation polls a postgres create operation, polling again when a poll fails with a
// transient error (observed: 500 INTERNAL_ERROR and inactivity timeouts on GET .../operations/...).
// The create request was already accepted, so polling again cannot create a second resource,
// whereas a failure returned from DoCreate is not retried at all.
func waitCreateOperation[T any](ctx context.Context, wait func(context.Context, ...api.Option) (T, error)) (T, error) {
	return retry.With(ctx, retry.IsTransientRead, func() (T, error) {
		return wait(ctx)
	})
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
