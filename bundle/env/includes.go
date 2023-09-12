package env

import "context"

// IncludesVariable names the environment variable that holds additional configuration paths to include
// during bundle configuration loading. Also see `bundle/config/mutator/process_root_includes.go`.
const IncludesVariable = "DATABRICKS_BUNDLE_INCLUDES"

// Includes returns the bundle Includes environment variable.
func Includes(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		IncludesVariable,
	})
}
