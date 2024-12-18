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

// Changes into specified directory for the duration of the test.
// Returns the current working directory.
func Chdir(t TestingT, dir string) string {
	// Prevent parallel execution when changing the working directory.
	// t.Setenv automatically fails if t.Parallel is set.
	t.Setenv("DO_NOT_RUN_IN_PARALLEL", "true")

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

func InsertPathEntry(t TestingT, path string) {
	var separator string
	if runtime.GOOS == "windows" {
		separator = ";"
	} else {
		separator = ":"
	}

	t.Setenv("PATH", path+separator+os.Getenv("PATH"))
}

func InsertVirtualenvInPath(t TestingT, venvPath string) {
	if runtime.GOOS == "windows" {
		// https://github.com/pypa/virtualenv/commit/993ba1316a83b760370f5a3872b3f5ef4dd904c1
		venvPath = filepath.Join(venvPath, "Scripts")
	} else {
		venvPath = filepath.Join(venvPath, "bin")
	}

	InsertPathEntry(t, venvPath)
}
