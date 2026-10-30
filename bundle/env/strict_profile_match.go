package env

import (
	"context"

	envlib "github.com/databricks/cli/libs/env"
)

// StrictProfileMatchVariable names the environment variable that enables strict
// profile matching. When it is true, a bundle target with a host or workspace ID
// must match exactly one workspace profile in the config file, even if
// credentials are also set in the environment. A profile without a workspace_id
// matches only targets without a workspace ID. Every section with a host counts,
// including [DEFAULT]. A target with a workspace ID but no host uses the profile
// from --profile, workspace.profile or [__settings__].default_profile instead. A
// target with neither resolves as without strict mode.
const StrictProfileMatchVariable = "DATABRICKS_BUNDLE_STRICT_PROFILE_MATCH"

// StrictProfileMatch reports whether strict profile matching is enabled. The
// value is read as a boolean, so "0" or "false" leave it off.
func StrictProfileMatch(ctx context.Context) bool {
	enabled, _ := envlib.GetBool(ctx, StrictProfileMatchVariable)
	return enabled
}
