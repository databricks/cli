package direct

import (
	"errors"

	"github.com/databricks/databricks-sdk-go/apierr"
)

// isManagedByParent reports whether err is an API error carrying the
// declarative_context=MANAGED_BY_PARENT marker in ErrorInfo.metadata. The
// server uses this to signal that a resource's lifecycle is owned by a
// parent (e.g. a Lakebase RW endpoint inside a branch, or a root branch
// inside a project) and the standalone Delete can be safely disregarded —
// the parent's Delete will cascade-clean. Mirrors the TF provider's
// declarative.IsDeleteError suppression.
func isManagedByParent(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok || apiErr == nil {
		return false
	}
	info := apiErr.ErrorDetails().ErrorInfo
	return info != nil && info.Metadata["declarative_context"] == "MANAGED_BY_PARENT"
}
