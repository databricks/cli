package lock

import (
	"context"

	"github.com/databricks/cli/libs/filer"
)

// IncrementDeploymentVersion is exported for testing.
var IncrementDeploymentVersion = func(ctx context.Context, f filer.Filer) (int64, error) {
	return incrementDeploymentVersion(ctx, f)
}
