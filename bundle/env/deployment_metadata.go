package env

import (
	"context"

	envlib "github.com/databricks/cli/libs/env"
)

// managedStateVariable names the environment variable that controls whether
// server-managed state (via the deployment metadata service) is used for
// locking and, in a future change, resource state management.
//
// The variable is treated as a boolean and accepts the usual spellings:
// "true"/"false", "1"/"0", "yes"/"no", "on"/"off" (case-insensitive). An
// empty or absent value falls back to the historical filesystem-based
// behavior.
const managedStateVariable = "DATABRICKS_BUNDLE_MANAGED_STATE"

// ManagedState returns the raw value of DATABRICKS_BUNDLE_MANAGED_STATE if
// set. Callers that only need a bool should use IsManagedState.
func ManagedState(ctx context.Context) (string, bool) {
	return get(ctx, []string{managedStateVariable})
}

// IsManagedState reports whether the DATABRICKS_BUNDLE_MANAGED_STATE
// environment variable is set to a truthy value.
func IsManagedState(ctx context.Context) bool {
	v, ok := envlib.GetBool(ctx, managedStateVariable)
	return ok && v
}
