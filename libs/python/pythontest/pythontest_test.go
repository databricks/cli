package pythontest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVenvSuccess(t *testing.T) {
	// Test at least two version to ensure we capture a case where venv version does not match system one
	for _, pythonVersion := range []string{"3.11", "3.12"} {
		t.Run(pythonVersion, func(t *testing.T) {
			ctx := context.Background()
			opts := VenvOpts{PythonVersion: pythonVersion}
			RequireActivatedPythonEnv(t, ctx, &opts)
			require.DirExists(t, opts.EnvPath)
			require.DirExists(t, opts.BinPath)
			require.FileExists(t, opts.PythonExe)
		})
	}
}

func TestWrongVersion(t *testing.T) {
	require.Error(t, CreatePythonEnv(&VenvOpts{PythonVersion: "4.0"}))
}

func TestMissingVersion(t *testing.T) {
	require.Error(t, CreatePythonEnv(nil))
	require.Error(t, CreatePythonEnv(&VenvOpts{}))
}
