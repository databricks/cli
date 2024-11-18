package dbr

import (
	"context"
	"os"
	"runtime"

	"github.com/databricks/cli/libs/env"
)

// Dereference [os.Stat] to allow mocking in tests.
var statFunc = os.Stat

// detect returns true if the current process is running on a Databricks Runtime.
// Its return value is meant to be cached in the context.
func detect(ctx context.Context) bool {
	// Databricks Runtime implies Linux.
	// Return early on other operating systems.
	if runtime.GOOS != "linux" {
		return false
	}

	// Databricks Runtime always has the DATABRICKS_RUNTIME_VERSION environment variable set.
	if value, ok := env.Lookup(ctx, "DATABRICKS_RUNTIME_VERSION"); !ok || value == "" {
		return false
	}

	// Expect to see a "/databricks" directory.
	if fi, err := statFunc("/databricks"); err != nil || !fi.IsDir() {
		return false
	}

	// All checks passed.
	return true
}
