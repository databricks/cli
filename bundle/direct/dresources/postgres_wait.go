package dresources

import (
	"context"

	"github.com/databricks/cli/libs/retry"
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
