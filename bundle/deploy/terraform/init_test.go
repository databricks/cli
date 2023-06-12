package terraform

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func unsetEnv(t *testing.T, name string) {
	t.Setenv(name, "")
	err := os.Unsetenv(name)
	require.NoError(t, err)
}

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

	err = Initialize().Apply(context.Background(), bundle)
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
	t.Setenv("TMPDIR", "/foo/bar")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(env, b)
	require.NoError(t, err)

	// Assert that we pass through TMPDIR.
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
	unsetEnv(t, "TMPDIR")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(env, b)
	require.NoError(t, err)

	// Assert that we don't pass through TMPDIR.
	assert.Equal(t, map[string]string{}, env)
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
	t.Setenv("TMP", "c:\\foo\\a")
	t.Setenv("TEMP", "c:\\foo\\b")
	t.Setenv("USERPROFILE", "c:\\foo\\c")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(env, b)
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
	unsetEnv(t, "TMP")
	t.Setenv("TEMP", "c:\\foo\\b")
	t.Setenv("USERPROFILE", "c:\\foo\\c")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(env, b)
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
	unsetEnv(t, "TMP")
	unsetEnv(t, "TEMP")
	t.Setenv("USERPROFILE", "c:\\foo\\c")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(env, b)
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
	unsetEnv(t, "TMP")
	unsetEnv(t, "TEMP")
	unsetEnv(t, "USERPROFILE")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(env, b)
	require.NoError(t, err)

	// assert TMP is set to b.CacheDir("tmp")
	tmpDir, err := b.CacheDir("tmp")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"TMP": tmpDir,
	}, env)
}

func TestSetProxyEnvVars(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Environment: "whatever",
			},
		},
	}

	// Temporarily clear environment variables.
	clearEnv := func() {
		for _, v := range []string{"http_proxy", "https_proxy", "no_proxy"} {
			for _, v := range []string{strings.ToUpper(v), strings.ToLower(v)} {
				t.Setenv(v, "foo")
				os.Unsetenv(v)
			}
		}
	}

	// No proxy env vars set.
	clearEnv()
	env := make(map[string]string, 0)
	err := setProxyEnvVars(env, b)
	require.NoError(t, err)
	assert.Len(t, env, 0)

	// Lower case set.
	clearEnv()
	t.Setenv("http_proxy", "foo")
	t.Setenv("https_proxy", "foo")
	t.Setenv("no_proxy", "foo")
	env = make(map[string]string, 0)
	err = setProxyEnvVars(env, b)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"http_proxy", "https_proxy", "no_proxy"}, maps.Keys(env))

	// Upper case set.
	clearEnv()
	t.Setenv("HTTP_PROXY", "foo")
	t.Setenv("HTTPS_PROXY", "foo")
	t.Setenv("NO_PROXY", "foo")
	env = make(map[string]string, 0)
	err = setProxyEnvVars(env, b)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}, maps.Keys(env))
}
