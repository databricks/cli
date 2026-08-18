package env

import "context"

// ResourceMaxWaitVariable names the environment variable that caps how long deployment waits
// for a resource to reach its target state.
const ResourceMaxWaitVariable = "DATABRICKS_BUNDLE_RESOURCE_MAX_WAIT"

// ResourceMaxWait returns the cap (in seconds) on waiting for a resource to reach its target
// state.
func ResourceMaxWait(ctx context.Context) (string, bool) {
	return get(ctx, []string{ResourceMaxWaitVariable})
}
