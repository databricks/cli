package dresources

import (
	"context"

	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go/experimental/api"
)

// postgresSpecUpdateMask masks the whole spec on update, with the full desired spec as
// the body. Probed against a real workspace on 2026-11-19, the API then applies what the
// body carries and keeps the stored value of most fields the body omits, so a field
// removed from the config usually keeps its last value. A field mask per changed path
// is rejected instead when that path's value is omitted ("is in update_mask but not
// provided in request").
var postgresSpecUpdateMask = []string{"spec"}

// waitCreateOperation polls a postgres create operation, polling again when a poll fails with a
// transient error (observed: 500 INTERNAL_ERROR and inactivity timeouts on GET .../operations/...).
// The create request was already accepted, so polling again cannot create a second resource,
// whereas a failure returned from DoCreate is not retried at all.
func waitCreateOperation[T any](ctx context.Context, wait func(context.Context, ...api.Option) (T, error)) (T, error) {
	return retry.With(ctx, retry.IsTransientRead, func() (T, error) {
		return wait(ctx)
	})
}
