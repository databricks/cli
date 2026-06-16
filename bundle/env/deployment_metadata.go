package env

import "context"

// managedStateVariable names the environment variable that controls whether
// server-managed state is used for locking and resource state management.
const managedStateVariable = "DATABRICKS_BUNDLE_MANAGED_STATE"

// ManagedState returns the environment variable that controls whether
// server-managed state is used for locking and resource state management.
func ManagedState(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		managedStateVariable,
	})
}
