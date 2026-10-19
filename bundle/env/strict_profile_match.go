package env

import "context"

// StrictProfileMatchVariable names the environment variable that enables strict
// profile matching. When it is set, the bundle target's host and workspace ID
// must match exactly one profile in the config file, even if credentials are
// also set in the environment. A profile without a workspace_id matches only
// targets without a workspace ID.
const StrictProfileMatchVariable = "DATABRICKS_BUNDLE_STRICT_PROFILE_MATCH"

// StrictProfileMatch returns the environment variable that enables strict profile matching.
func StrictProfileMatch(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		StrictProfileMatchVariable,
	})
}
