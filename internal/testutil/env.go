package testutil

import (
	"os"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/env"
)

// CleanupEnvironment sets up a pristine environment containing only $PATH and $HOME.
// The original environment is restored upon test completion.
// Note: use of this function is incompatible with parallel execution.
func CleanupEnvironment(t TestingT) {
	path := env.Get(t.Context(), "PATH")
	pwd := env.Get(t.Context(), "PWD")

	// Clear all environment variables.
	NullEnvironment(t)

	// We use t.Setenv instead of os.Setenv because the former actively
	// prevents a test being run with t.Parallel. Modifying the environment
	// within a test is not compatible with running tests in parallel
	// because of isolation; the environment is scoped to the process.
	t.Setenv("PATH", path)
	t.Setenv("HOME", pwd)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", pwd)
	}
}

// NullEnvironment sets up an empty environment with absolutely no environment variables set.
// The original environment is restored upon test completion.
// Note: use of this function is incompatible with parallel execution
func NullEnvironment(t TestingT) {
	// Restore environment when test finishes.
	environ := os.Environ()
	t.Cleanup(func() {
		// Restore original environment.
		for _, kv := range environ {
			kvs := strings.SplitN(kv, "=", 2)
			os.Setenv(kvs[0], kvs[1])
		}
	})

	os.Clearenv()
}
