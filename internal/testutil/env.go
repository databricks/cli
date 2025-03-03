package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/stretchr/testify/require"
)

// CleanupEnvironment sets up a pristine environment containing only $PATH and $HOME.
// The original environment is restored upon test completion.
// Note: use of this function is incompatible with parallel execution.
func CleanupEnvironment(t TestingT) {
	path := os.Getenv("PATH")
	pwd := os.Getenv("PWD")

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

// Changes into specified directory for the duration of the test.
// Returns the current working directory.
func Chdir(t TestingT, dir string) string {
	// Prevent parallel execution when changing the working directory.
	// t.Setenv automatically fails if t.Parallel is set.
	t.Setenv("DO_NOT_RUN_IN_PARALLEL", "true")

	wd, err := os.Getwd()
	require.NoError(t, err)
	if os.Getenv("TESTS_ORIG_WD") == "" {
		t.Setenv("TESTS_ORIG_WD", wd)
	}

	abs, err := filepath.Abs(dir)
	require.NoError(t, err)

	err = os.Chdir(abs)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	})

	return wd
}

// Return filename ff testutil.Chdir was not called.
// Return absolute path to filename testutil.Chdir() was called.
func TestData(filename string) string {
	// Note, if TESTS_ORIG_WD is not set, Getenv return "" and Join returns filename
	return filepath.Join(os.Getenv("TESTS_ORIG_WD"), filename)
}
