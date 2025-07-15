package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/hashicorp/go-version"
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
		BundleRootPath: t.TempDir(),
		Config: config.Root{
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

	diags := bundle.Apply(context.Background(), b, Initialize())
	require.NoError(t, diags.Error())
}

func TestSetTempDirEnvVarsForUnixWithTmpDirSet(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.SkipNow()
	}

	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
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
		BundleRootPath: t.TempDir(),
		Config: config.Root{
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
		BundleRootPath: t.TempDir(),
		Config: config.Root{
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
		BundleRootPath: t.TempDir(),
		Config: config.Root{
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
		BundleRootPath: t.TempDir(),
		Config: config.Root{
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

	// assert TMP is set to b.LocalStateDir("tmp")
	tmpDir, err := b.LocalStateDir(context.Background(), "tmp")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"TMP": tmpDir,
	}, env)
}

func TestSetProxyEnvVars(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
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
	assert.Empty(t, env)

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

func TestSetUserAgentExtra_Python(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Experimental: &config.Experimental{
				Python: config.Python{
					Resources: []string{"my_project.resources:load_resources"},
				},
			},
		},
	}

	env := make(map[string]string, 0)
	err := setUserAgentExtraEnvVar(env, b)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"DATABRICKS_USER_AGENT_EXTRA": "cli/0.0.0-dev databricks-pydabs/0.7.0",
	}, env)
}

func TestInheritEnvVars(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("PATH", "/foo:/bar")
	t.Setenv("TF_CLI_CONFIG_FILE", "/tmp/config.tfrc")
	t.Setenv("AZURE_CONFIG_DIR", "/tmp/foo/bar")

	ctx := context.Background()
	env := map[string]string{}
	err := inheritEnvVars(ctx, env)
	if assert.NoError(t, err) {
		assert.Equal(t, "/home/testuser", env["HOME"])
		assert.Equal(t, "/foo:/bar", env["PATH"])
		assert.Equal(t, "/tmp/config.tfrc", env["TF_CLI_CONFIG_FILE"])
		assert.Equal(t, "/tmp/foo/bar", env["AZURE_CONFIG_DIR"])
	}
}

func TestInheritOIDCTokenEnvCustom(t *testing.T) {
	t.Setenv("DATABRICKS_OIDC_TOKEN_ENV", "custom_DATABRICKS_OIDC_TOKEN")
	t.Setenv("custom_DATABRICKS_OIDC_TOKEN", "foobar")

	ctx := context.Background()
	env := map[string]string{}
	err := inheritEnvVars(ctx, env)
	require.NoError(t, err)
	assert.Equal(t, "foobar", env["custom_DATABRICKS_OIDC_TOKEN"])
	assert.Equal(t, "custom_DATABRICKS_OIDC_TOKEN", env["DATABRICKS_OIDC_TOKEN_ENV"])
}

func TestInheritOIDCTokenEnv(t *testing.T) {
	t.Setenv("DATABRICKS_OIDC_TOKEN", "foobar")

	ctx := context.Background()
	env := map[string]string{}
	err := inheritEnvVars(ctx, env)
	require.NoError(t, err)
	assert.Equal(t, "foobar", env["DATABRICKS_OIDC_TOKEN"])
	assert.Equal(t, "", env["DATABRICKS_OIDC_TOKEN_ENV"])
}

func TestSetUserProfileFromInheritEnvVars(t *testing.T) {
	t.Setenv("USERPROFILE", "c:\\foo\\c")

	env := make(map[string]string, 0)
	err := inheritEnvVars(context.Background(), env)
	require.NoError(t, err)

	assert.Contains(t, env, "USERPROFILE")
	assert.Equal(t, "c:\\foo\\c", env["USERPROFILE"])
}

func TestInheritEnvVarsWithAbsentTFConfigFile(t *testing.T) {
	ctx := context.Background()
	envMap := map[string]string{}
	ctx = env.Set(ctx, "DATABRICKS_TF_PROVIDER_VERSION", schema.ProviderVersion)
	ctx = env.Set(ctx, "DATABRICKS_TF_CLI_CONFIG_FILE", "/tmp/config.tfrc")
	err := inheritEnvVars(ctx, envMap)
	require.NoError(t, err)
	require.NotContains(t, envMap, "TF_CLI_CONFIG_FILE")
}

func TestInheritEnvVarsWithWrongTFProviderVersion(t *testing.T) {
	ctx := context.Background()
	envMap := map[string]string{}
	configFile := createTempFile(t, t.TempDir(), "config.tfrc", false)
	ctx = env.Set(ctx, "DATABRICKS_TF_PROVIDER_VERSION", "wrong")
	ctx = env.Set(ctx, "DATABRICKS_TF_CLI_CONFIG_FILE", configFile)
	err := inheritEnvVars(ctx, envMap)
	require.NoError(t, err)
	require.NotContains(t, envMap, "TF_CLI_CONFIG_FILE")
}

func TestInheritEnvVarsWithCorrectTFCLIConfigFile(t *testing.T) {
	ctx := context.Background()
	envMap := map[string]string{}
	configFile := createTempFile(t, t.TempDir(), "config.tfrc", false)
	ctx = env.Set(ctx, "DATABRICKS_TF_PROVIDER_VERSION", schema.ProviderVersion)
	ctx = env.Set(ctx, "DATABRICKS_TF_CLI_CONFIG_FILE", configFile)
	err := inheritEnvVars(ctx, envMap)
	require.NoError(t, err)
	require.Contains(t, envMap, "TF_CLI_CONFIG_FILE")
	require.Equal(t, configFile, envMap["TF_CLI_CONFIG_FILE"])
}

func createFakeTerraformBinary(t *testing.T, dest, version string) string {
	binPath := filepath.Join(dest, product.Terraform.BinaryName())
	jsonPayload := fmt.Sprintf(`{
  "terraform_version": "%s",
  "provider_selections": {}
}
`, version)

	if runtime.GOOS == "windows" {
		// Create stub with .exe that cannot be executed.
		// We need it because the implementation looks for this file to see if it is cached.
		createFakeTerraformBinaryWindows(t, binPath, jsonPayload, version)
		// Create stub with .bat that can be executed.
		// We use this path when setting the DATABRICKS_TF_EXEC_PATH environment variable.
		createFakeTerraformBinaryWindows(t, binPath+".bat", jsonPayload, version)
		// Return the .bat file that can be executed.
		return binPath + ".bat"
	} else {
		return createFakeTerraformBinaryOther(t, binPath, jsonPayload, version)
	}
}

func createFakeTerraformBinaryWindows(t *testing.T, binPath, jsonPayload, version string) string {
	f, err := os.Create(binPath)
	require.NoError(t, err)
	defer func() {
		err = f.Close()
		require.NoError(t, err)
	}()
	// Write the payload to a temp file and type it, to avoid escaping issues
	tmpJsonPath := filepath.Join(t.TempDir(), "payload.json")
	err = os.WriteFile(tmpJsonPath, []byte(jsonPayload), 0o644)
	require.NoError(t, err)
	_, err = f.WriteString(fmt.Sprintf(`@echo off
REM This is a fake Terraform binary that returns the JSON payload.
REM It stubs version %s.
type "%s"
`, version, tmpJsonPath))
	require.NoError(t, err)
	return binPath
}

func createFakeTerraformBinaryOther(t *testing.T, binPath, jsonPayload, version string) string {
	f, err := os.Create(binPath)
	require.NoError(t, err)
	defer func() {
		err = f.Close()
		require.NoError(t, err)
	}()
	err = f.Chmod(0o777)
	require.NoError(t, err)
	_, err = f.WriteString(fmt.Sprintf(`#!/bin/sh
# This is a fake Terraform binary that returns the JSON payload.
# It stubs version %s.
cat <<EOF
%sEOF
`, version, jsonPayload))
	require.NoError(t, err)
	return binPath
}

type testInstaller struct {
	t *testing.T
}

func (i testInstaller) Install(ctx context.Context, dir string, version *version.Version) (string, error) {
	return createFakeTerraformBinary(i.t, dir, version.String()), nil
}

func TestFindExecPath_NoBinary(t *testing.T) {
	ctx := context.Background()
	m := &initialize{}
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}

	// Verify that the binary for the default version is downloaded.
	_, err := m.findExecPath(ctx, b, b.Config.Bundle.Terraform, testInstaller{t})
	require.NoError(t, err)
	assert.Contains(t, testutil.ReadFile(t, b.Config.Bundle.Terraform.ExecPath), "1.5.5")
}

func TestFindExecPath_UseExistingBinary(t *testing.T) {
	ctx := context.Background()
	m := &initialize{}
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}

	// Create a pre-existing Terraform binary to avoid downloading it
	cacheDir, _ := b.LocalStateDir(ctx, "bin")
	createFakeTerraformBinary(t, cacheDir, "1.2.3")

	// Verify that the pre-existing Terraform binary is used.
	_, err := m.findExecPath(ctx, b, b.Config.Bundle.Terraform, testInstaller{t})
	require.NoError(t, err)
	assert.Contains(t, testutil.ReadFile(t, b.Config.Bundle.Terraform.ExecPath), "1.2.3")
}

func TestFindExecPath_ExecPathWrongVersion(t *testing.T) {
	ctx := context.Background()
	m := &initialize{}
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}

	// Configure a valid exec path.
	version := "1.2.4"
	execPath := createFakeTerraformBinary(t, t.TempDir(), version)
	ctx = env.Set(ctx, "DATABRICKS_TF_EXEC_PATH", execPath)

	// Verify that the error is returned.
	expected := []string{
		`Terraform binary at ` + execPath + ` (from $DATABRICKS_TF_EXEC_PATH) is ` + version + ` but expected version is ` + defaultTerraformVersion.Version.String() + `.`,
		`Set DATABRICKS_TF_VERSION to ` + version + ` to continue.`,
	}
	_, err := m.findExecPath(ctx, b, b.Config.Bundle.Terraform, testInstaller{t})
	require.ErrorContains(t, err, strings.Join(expected, " "))
}

func TestFindExecPath_ExecPathMatchingVersion(t *testing.T) {
	ctx := context.Background()
	m := &initialize{}
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}

	// Configure a valid exec path.
	execPath := createFakeTerraformBinary(t, t.TempDir(), defaultTerraformVersion.Version.String())
	ctx = env.Set(ctx, "DATABRICKS_TF_EXEC_PATH", execPath)

	// Verify that the pre-existing Terraform binary is used.
	_, err := m.findExecPath(ctx, b, b.Config.Bundle.Terraform, testInstaller{t})
	require.NoError(t, err)
	assert.Equal(t, execPath, b.Config.Bundle.Terraform.ExecPath)
}

func TestFindExecPath_Version_NoExecPath(t *testing.T) {
	ctx := context.Background()
	m := &initialize{}
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}

	// Configure a fake version.
	version := "1.2.3"
	ctx = env.Set(ctx, "DATABRICKS_TF_VERSION", version)

	// Verify that the binary for the default version is downloaded.
	_, err := m.findExecPath(ctx, b, b.Config.Bundle.Terraform, testInstaller{t})
	require.NoError(t, err)
	assert.Contains(t, testutil.ReadFile(t, b.Config.Bundle.Terraform.ExecPath), version)
}

func TestFindExecPath_Version_ExecPathBadFile(t *testing.T) {
	ctx := context.Background()
	m := &initialize{}
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}

	// Configure a fake version.
	version := "1.2.3"
	ctx = env.Set(ctx, "DATABRICKS_TF_VERSION", version)

	// Configure an invalid exec path.
	ctx = env.Set(ctx, "DATABRICKS_TF_EXEC_PATH", "/tmp/terraform")

	// Verify that the error is returned.
	_, err := m.findExecPath(ctx, b, b.Config.Bundle.Terraform, testInstaller{t})
	require.ErrorContains(t, err, "unable to execute DATABRICKS_TF_EXEC_PATH:")
}

func TestFindExecPath_Version_ExecPathWrongVersion(t *testing.T) {
	ctx := context.Background()
	m := &initialize{}
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}

	// Configure a fake version.
	version := "1.2.3"
	ctx = env.Set(ctx, "DATABRICKS_TF_VERSION", version)

	// Configure a valid exec path.
	execPath := createFakeTerraformBinary(t, t.TempDir(), "1.2.4")
	ctx = env.Set(ctx, "DATABRICKS_TF_EXEC_PATH", execPath)

	// Verify that the error is returned.
	expected := []string{
		`Terraform binary at ` + execPath + ` (from $DATABRICKS_TF_EXEC_PATH) is 1.2.4 but expected version is 1.2.3 (from $DATABRICKS_TF_VERSION).`,
		`Update $DATABRICKS_TF_EXEC_PATH and $DATABRICKS_TF_VERSION so that versions match.`,
	}
	_, err := m.findExecPath(ctx, b, b.Config.Bundle.Terraform, testInstaller{t})
	require.ErrorContains(t, err, strings.Join(expected, " "))
}

func TestFindExecPath_Version_ExecPathMatchingVersion(t *testing.T) {
	ctx := context.Background()
	m := &initialize{}
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target:    "whatever",
				Terraform: &config.Terraform{},
			},
		},
	}

	// Configure a fake version.
	otherVersion := "1.2.3"
	ctx = env.Set(ctx, "DATABRICKS_TF_VERSION", otherVersion)

	// Configure a valid exec path.
	execPath := createFakeTerraformBinary(t, t.TempDir(), otherVersion)
	ctx = env.Set(ctx, "DATABRICKS_TF_EXEC_PATH", execPath)

	// Verify that the specified Terraform binary is used.
	_, err := m.findExecPath(ctx, b, b.Config.Bundle.Terraform, testInstaller{t})
	require.NoError(t, err)
	require.Equal(t, execPath, b.Config.Bundle.Terraform.ExecPath)
}

func createTempFile(t *testing.T, dest, name string, executable bool) string {
	binPath := filepath.Join(dest, name)
	f, err := os.Create(binPath)
	require.NoError(t, err)
	defer func() {
		err = f.Close()
		require.NoError(t, err)
	}()
	if executable {
		err = f.Chmod(0o777)
		require.NoError(t, err)
	}
	return binPath
}

func TestGetEnvVarWithMatchingVersion(t *testing.T) {
	envVarName := "FOO"
	versionVarName := "FOO_VERSION"

	tmp := t.TempDir()
	file := testutil.Touch(t, tmp, "bar")

	tc := []struct {
		envValue       string
		versionValue   string
		currentVersion string
		expected       string
	}{
		{
			envValue:       file,
			versionValue:   "1.2.3",
			currentVersion: "1.2.3",
			expected:       file,
		},
		{
			envValue:       "does-not-exist",
			versionValue:   "1.2.3",
			currentVersion: "1.2.3",
			expected:       "",
		},
		{
			envValue:       file,
			versionValue:   "1.2.3",
			currentVersion: "1.2.4",
			expected:       "",
		},
		{
			envValue:       "",
			versionValue:   "1.2.3",
			currentVersion: "1.2.3",
			expected:       "",
		},
		{
			envValue:       file,
			versionValue:   "",
			currentVersion: "1.2.3",
			expected:       file,
		},
	}

	for _, c := range tc {
		t.Run("", func(t *testing.T) {
			t.Setenv(envVarName, c.envValue)
			t.Setenv(versionVarName, c.versionValue)

			actual, err := getEnvVarWithMatchingVersion(context.Background(), envVarName, versionVarName, c.currentVersion)
			require.NoError(t, err)
			assert.Equal(t, c.expected, actual)
		})
	}
}
