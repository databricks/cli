package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// CleanupEnvironment sets up a pristine environment containing only $PATH and $HOME.
// The original environment is restored upon test completion.
// Note: use of this function is incompatible with parallel execution.
func CleanupEnvironment(t *testing.T) {
	// Restore environment when test finishes.
	environ := os.Environ()
	t.Cleanup(func() {
		// Restore original environment.
		for _, kv := range environ {
			kvs := strings.SplitN(kv, "=", 2)
			os.Setenv(kvs[0], kvs[1])
		}
	})

	path := os.Getenv("PATH")
	pwd := os.Getenv("PWD")
	os.Clearenv()

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

// GetEnvOrSkipTest proceeds with test only with that env variable
func GetEnvOrSkipTest(t *testing.T, name string) string {
	value := os.Getenv(name)
	if value == "" {
		t.Skipf("Environment variable %s is missing", name)
	}
	return value
}

// Changes into specified directory for the duration of the test.
// Returns the current working directory.
func Chdir(t *testing.T, dir string) string {
	wd, err := os.Getwd()
	require.NoError(t, err)

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
