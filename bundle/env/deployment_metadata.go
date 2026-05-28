package env

import (
	"context"

	envlib "github.com/databricks/cli/libs/env"
)

// managedStateVariable names the environment variable that opts a bundle into
// the deployment metadata service (DMS) for locking and resource-state
// management. Defaults to the historical filesystem-based behavior when
// unset.
//
// The variable is treated as a boolean and accepts the usual spellings:
// "true"/"false", "1"/"0", "yes"/"no", "on"/"off" (case-insensitive).
const managedStateVariable = "DATABRICKS_BUNDLE_MANAGED_STATE"

// IsManagedState reports whether the DATABRICKS_BUNDLE_MANAGED_STATE
// environment variable is set to a truthy value.
func IsManagedState(ctx context.Context) bool {
	v, ok := envlib.GetBool(ctx, managedStateVariable)
	return ok && v
}
