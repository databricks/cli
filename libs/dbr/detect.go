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
func detect(ctx context.Context) Environment {
	// Databricks Runtime implies Linux.
	// Return early on other operating systems.
	if runtime.GOOS != "linux" {
		return Environment{}
	}

	// Databricks Runtime always has the DATABRICKS_RUNTIME_VERSION environment variable set.
	version, ok := env.Lookup(ctx, "DATABRICKS_RUNTIME_VERSION")
	if !ok || version == "" {
		return Environment{}
	}

	// Expect to see a "/databricks" directory.
	if fi, err := statFunc("/databricks"); err != nil || !fi.IsDir() {
		return Environment{}
	}

	// All checks passed.
	return Environment{
		IsDbr:   true,
		Version: version,
	}
}
