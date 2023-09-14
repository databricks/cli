package env

import "context"

// TempDirVariable names the environment variable that holds the temporary directory to use.
const TempDirVariable = "DATABRICKS_BUNDLE_TMP"

// TempDir returns the temporary directory to use.
func TempDir(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		TempDirVariable,
	})
}
