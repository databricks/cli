package terraform

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitEnvironmentVariables(t *testing.T) {
	_, err := exec.LookPath("terraform")
	if err != nil {
		t.Skipf("cannot find terraform binary: %s", err)
	}

	bundle := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Environment: "whatever",
				Terraform: &config.Terraform{
					ExecPath: "terraform",
				},
			},
		},
	}

	// Trigger initialization of workspace client.
	// TODO(pietern): create test fixture that initializes a mocked client.
	t.Setenv("DATABRICKS_HOST", "https://x")
	t.Setenv("DATABRICKS_TOKEN", "foobar")
	bundle.WorkspaceClient()

	_, err = Initialize().Apply(context.Background(), bundle)
	require.NoError(t, err)
}

func TestSetTempDirEnvVarsForUnixWithTmpDirSet(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.SkipNow()
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Environment: "whatever",
			},
		},
	}

	// Set TMPDIR environment variable
	err := os.Setenv("TMPDIR", "/foo/bar")
	defer os.Unsetenv("TMPDIR")
	require.NoError(t, err)

	// compute env
	env := make(map[string]string, 0)
	err = setTempDirEnvVars(env, b)
	require.NoError(t, err)

	// assert that we pass through env var value
	assert.Equal(t, map[string]string{
		"TMPDIR": "/foo/bar",
	}, env)
}

func TestSetTempDirEnvVarsForUnixWithTmpDirNotSet(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.SkipNow()
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Environment: "whatever",
			},
		},
	}

	// Unset TMPDIR environment variable confirm it's not set
	err := os.Unsetenv("TMPDIR")
	require.NoError(t, err)

	// compute env
	env := make(map[string]string, 0)
	err = setTempDirEnvVars(env, b)
	require.NoError(t, err)

	// assert tmp dir is set to b.CacheDir("tmp")
	tmpDir, err := b.CacheDir("tmp")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"TMPDIR": tmpDir,
	}, env)
}

func TestSetTempDirEnvVarsForWindowWithAllTmpDirEnvVarsSet(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Environment: "whatever",
			},
		},
	}

	// Set environment variables
	err := os.Setenv("TMP", "c:\\foo\\a")
	require.NoError(t, err)
	defer os.Unsetenv("TMP")
	err = os.Setenv("TEMP", "c:\\foo\\b")
	require.NoError(t, err)
	defer os.Unsetenv("TEMP")
	err = os.Setenv("USERPROFILE", "c:\\foo\\c")
	require.NoError(t, err)
	defer os.Unsetenv("USERPROFILE")

	// compute env
	env := make(map[string]string, 0)
	err = setTempDirEnvVars(env, b)
	require.NoError(t, err)

	// assert that we pass through the highest priority env var value
	assert.Equal(t, map[string]string{
		"TMP": "c:\\foo\\a",
	}, env)
}

func TestSetTempDirEnvVarsForWindowWithUserProfileAndTempSet(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Environment: "whatever",
			},
		},
	}

	// Set environment variables
	err := os.Unsetenv("TMP")
	require.NoError(t, err)
	err = os.Setenv("TEMP", "c:\\foo\\b")
	require.NoError(t, err)
	defer os.Unsetenv("TEMP")
	err = os.Setenv("USERPROFILE", "c:\\foo\\c")
	require.NoError(t, err)
	defer os.Unsetenv("USERPROFILE")

	// compute env
	env := make(map[string]string, 0)
	err = setTempDirEnvVars(env, b)
	require.NoError(t, err)

	// assert that we pass through the highest priority env var value
	assert.Equal(t, map[string]string{
		"TEMP": "c:\\foo\\b",
	}, env)
}

func TestSetTempDirEnvVarsForWindowWithUserProfileSet(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Environment: "whatever",
			},
		},
	}

	// Set environment variables
	err := os.Unsetenv("TMP")
	require.NoError(t, err)
	err = os.Unsetenv("TEMP")
	require.NoError(t, err)
	err = os.Setenv("USERPROFILE", "c:\\foo\\c")
	require.NoError(t, err)
	defer os.Unsetenv("USERPROFILE")

	// compute env
	env := make(map[string]string, 0)
	err = setTempDirEnvVars(env, b)
	require.NoError(t, err)

	// assert that we pass through the user profile
	assert.Equal(t, map[string]string{
		"USERPROFILE": "c:\\foo\\c",
	}, env)
}

func TestSetTempDirEnvVarsForWindowsWithoutAnyTempDirEnvVarsSet(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Environment: "whatever",
			},
		},
	}

	// unset all env vars
	err := os.Unsetenv("TMP")
	require.NoError(t, err)
	err = os.Unsetenv("TEMP")
	require.NoError(t, err)
	err = os.Unsetenv("USERPROFILE")
	require.NoError(t, err)

	// compute env
	env := make(map[string]string, 0)
	err = setTempDirEnvVars(env, b)
	require.NoError(t, err)

	// assert TMP is set to b.CacheDir("tmp")
	tmpDir, err := b.CacheDir("tmp")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"TMP": tmpDir,
	}, env)
}
