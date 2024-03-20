package terraform

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/hashicorp/hc-install/product"
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

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "whatever",
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
	b.WorkspaceClient()

	err = bundle.Apply(context.Background(), b, Initialize())
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
				Target: "whatever",
			},
		},
	}

	// Set TMPDIR environment variable
	t.Setenv("TMPDIR", "/foo/bar")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(context.Background(), env, b)
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
				Target: "whatever",
			},
		},
	}

	// Unset TMPDIR environment variable confirm it's not set
	unsetEnv(t, "TMPDIR")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(context.Background(), env, b)
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
				Target: "whatever",
			},
		},
	}

	// Set environment variables
	t.Setenv("TMP", "c:\\foo\\a")
	t.Setenv("TEMP", "c:\\foo\\b")
	t.Setenv("USERPROFILE", "c:\\foo\\c")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(context.Background(), env, b)
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
				Target: "whatever",
			},
		},
	}

	// Set environment variables
	unsetEnv(t, "TMP")
	t.Setenv("TEMP", "c:\\foo\\b")
	t.Setenv("USERPROFILE", "c:\\foo\\c")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(context.Background(), env, b)
	require.NoError(t, err)

	// assert that we pass through the highest priority env var value
	assert.Equal(t, map[string]string{
		"TEMP": "c:\\foo\\b",
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
				Target: "whatever",
			},
		},
	}

	// unset all env vars
	unsetEnv(t, "TMP")
	unsetEnv(t, "TEMP")
	unsetEnv(t, "USERPROFILE")

	// compute env
	env := make(map[string]string, 0)
	err := setTempDirEnvVars(context.Background(), env, b)
	require.NoError(t, err)

	// assert TMP is set to b.CacheDir("tmp")
	tmpDir, err := b.CacheDir(context.Background(), "tmp")
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
				Target: "whatever",
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
	err := setProxyEnvVars(context.Background(), env, b)
	require.NoError(t, err)
	assert.Len(t, env, 0)

	// Lower case set.
	clearEnv()
	t.Setenv("http_proxy", "foo")
	t.Setenv("https_proxy", "foo")
	t.Setenv("no_proxy", "foo")
	env = make(map[string]string, 0)
	err = setProxyEnvVars(context.Background(), env, b)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}, maps.Keys(env))

	// Upper case set.
	clearEnv()
	t.Setenv("HTTP_PROXY", "foo")
	t.Setenv("HTTPS_PROXY", "foo")
	t.Setenv("NO_PROXY", "foo")
	env = make(map[string]string, 0)
	err = setProxyEnvVars(context.Background(), env, b)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}, maps.Keys(env))
}

func TestInheritEnvVars(t *testing.T) {
	env := map[string]string{}

	t.Setenv("HOME", "/home/testuser")
	t.Setenv("PATH", "/foo:/bar")
	t.Setenv("TF_CLI_CONFIG_FILE", "/tmp/config.tfrc")

	err := inheritEnvVars(context.Background(), env)

	require.NoError(t, err)

	require.Equal(t, env["HOME"], "/home/testuser")
	require.Equal(t, env["PATH"], "/foo:/bar")
	require.Equal(t, env["TF_CLI_CONFIG_FILE"], "/tmp/config.tfrc")
}

func TestSetUserProfileFromInheritEnvVars(t *testing.T) {
	t.Setenv("USERPROFILE", "c:\\foo\\c")

	env := make(map[string]string, 0)
	err := inheritEnvVars(context.Background(), env)
	require.NoError(t, err)

	assert.Contains(t, env, "USERPROFILE")
	assert.Equal(t, env["USERPROFILE"], "c:\\foo\\c")
}

func TestInheritEnvVarsWithAbsentPluginsCacheDir(t *testing.T) {
	env := map[string]string{}
	t.Setenv("DATABRICKS_TF_PLUGIN_CACHE_DIR", "/tmp/cache")
	err := inheritEnvVars(context.Background(), env)
	require.NoError(t, err)
	require.NotContains(t, env, "TF_PLUGIN_CACHE_DIR")
}

func TestInheritEnvVarsWithRealPluginsCacheDir(t *testing.T) {
	env := map[string]string{}
	dir := t.TempDir()
	t.Setenv("DATABRICKS_TF_PLUGIN_CACHE_DIR", dir)
	err := inheritEnvVars(context.Background(), env)
	require.NoError(t, err)
	require.Equal(t, dir, env["TF_PLUGIN_CACHE_DIR"])
}

func createTerraformBinary(t *testing.T, dest string, name string) string {
	binPath := filepath.Join(dest, name)
	f, err := os.Create(binPath)
	require.NoError(t, err)
	defer func() {
		err = f.Close()
		require.NoError(t, err)
	}()
	err = f.Chmod(0755)
	require.NoError(t, err)
	return binPath
}

func TestFindExecPathFromEnvironmentWithWrongVersion(t *testing.T) {
	m := &initialize{}
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}
	// Create a pre-existing terraform bin to avoid downloading it
	cacheDir, _ := b.CacheDir(context.Background(), "bin")
	existingExecPath := createTerraformBinary(t, cacheDir, product.Terraform.BinaryName())
	// Create a new terraform binary and expose it through env vars
	tmpBinPath := createTerraformBinary(t, t.TempDir(), "terraform-bin")
	t.Setenv("DATABRICKS_TF_VERSION", "1.2.3")
	t.Setenv("DATABRICKS_TF_EXEC_PATH", tmpBinPath)
	_, err := m.findExecPath(context.Background(), b, b.Config.Bundle.Terraform)
	require.NoError(t, err)
	require.Equal(t, existingExecPath, b.Config.Bundle.Terraform.ExecPath)
}

func TestFindExecPathFromEnvironmentWithCorrectVersionAndNoBinary(t *testing.T) {
	m := &initialize{}
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}
	// Create a pre-existing terraform bin to avoid downloading it
	cacheDir, _ := b.CacheDir(context.Background(), "bin")
	existingExecPath := createTerraformBinary(t, cacheDir, product.Terraform.BinaryName())

	t.Setenv("DATABRICKS_TF_VERSION", TerraformVersion.String())
	t.Setenv("DATABRICKS_TF_EXEC_PATH", "/tmp/terraform")
	_, err := m.findExecPath(context.Background(), b, b.Config.Bundle.Terraform)
	require.NoError(t, err)
	require.Equal(t, existingExecPath, b.Config.Bundle.Terraform.ExecPath)
}

func TestFindExecPathFromEnvironmentWithCorrectVersionAndBinaryAndAlreadySetExecPath(t *testing.T) {
	m := &initialize{}
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}
	existingExecPath := createTerraformBinary(t, t.TempDir(), "terraform-existing")
	b.Config.Bundle.Terraform.ExecPath = existingExecPath
	// Create a new terraform binary and expose it through env vars
	tmpBinPath := createTerraformBinary(t, t.TempDir(), "terraform-bin")
	t.Setenv("DATABRICKS_TF_VERSION", TerraformVersion.String())
	t.Setenv("DATABRICKS_TF_EXEC_PATH", tmpBinPath)
	_, err := m.findExecPath(context.Background(), b, b.Config.Bundle.Terraform)
	require.NoError(t, err)
	require.Equal(t, existingExecPath, b.Config.Bundle.Terraform.ExecPath)
}

func TestFindExecPathFromEnvironmentWithCorrectVersionAndBinary(t *testing.T) {
	m := &initialize{}
	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}
	// Create a pre-existing terraform bin to avoid downloading it
	cacheDir, _ := b.CacheDir(context.Background(), "bin")
	createTerraformBinary(t, cacheDir, product.Terraform.BinaryName())
	// Create a new terraform binary and expose it through env vars
	tmpBinPath := createTerraformBinary(t, t.TempDir(), "terraform-bin")
	t.Setenv("DATABRICKS_TF_VERSION", TerraformVersion.String())
	t.Setenv("DATABRICKS_TF_EXEC_PATH", tmpBinPath)
	_, err := m.findExecPath(context.Background(), b, b.Config.Bundle.Terraform)
	require.NoError(t, err)
	require.Equal(t, tmpBinPath, b.Config.Bundle.Terraform.ExecPath)
}
