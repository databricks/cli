package env

import "context"

// TargetVariable names the environment variable that holds the ucm target to use.
const TargetVariable = "DATABRICKS_UCM_TARGET"

// Target returns the ucm target environment variable.
func Target(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		TargetVariable,
	})
}
