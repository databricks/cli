package env

import "context"

// RootVariable names the environment variable that holds the bundle root path.
const RootVariable = "DATABRICKS_BUNDLE_ROOT"

// Root returns the bundle root environment variable.
func Root(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		RootVariable,

		// Primary variable name for the bundle root until v0.204.0.
		"BUNDLE_ROOT",
	})
}
