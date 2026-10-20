package env

import "context"

// StrictProfileMatchVariable names the environment variable that enables strict
// profile matching. When it is set, the bundle target's host and workspace ID
// must match exactly one workspace profile in the config file, even if
// credentials are also set in the environment. A profile without a workspace_id
// matches only targets without a workspace ID. Every section with a host counts,
// including [DEFAULT]. A target without a host uses the profile from --profile,
// workspace.profile or [__settings__].default_profile instead.
const StrictProfileMatchVariable = "DATABRICKS_BUNDLE_STRICT_PROFILE_MATCH"

// StrictProfileMatch returns the environment variable that enables strict profile matching.
func StrictProfileMatch(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		StrictProfileMatchVariable,
	})
}
