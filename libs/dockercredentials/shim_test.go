package dockercredentials

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestDatabricksExecutable(t *testing.T, dir string) string {
	t.Helper()
	name := "databricks"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("databricks executable"), 0o755))
	return path
}

func TestShimFilename(t *testing.T) {
	assert.Equal(t, "docker-credential-databricks", shimFilename("linux"))
	assert.Equal(t, "docker-credential-databricks.cmd", shimFilename("windows"))
}

func TestInstallWindowsShimWritesCommandScript(t *testing.T) {
	installDir := t.TempDir()
	databricksPath := filepath.Join(installDir, "databricks.exe")
	require.NoError(t, os.WriteFile(databricksPath, []byte("databricks executable"), 0o755))
	t.Setenv("PATH", installDir)

	got, err := installShimForGOOS(databricksPath, "windows")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(installDir, "docker-credential-databricks.cmd"), got.Path)

	raw, err := os.ReadFile(got.Path)
	require.NoError(t, err)
	assert.NotEqual(t, "databricks executable", string(raw))
	assert.Contains(t, string(raw), "auth docker token")
}

func TestUnixShimExecutesOnlyGetAndForcesLogsToStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell shim test")
	}

	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args")
	envPath := filepath.Join(dir, "env")
	stdinPath := filepath.Join(dir, "stdin")
	fakeDir := filepath.Join(dir, "bin$DATABRICKS_LOG_FILE")
	require.NoError(t, os.MkdirAll(fakeDir, 0o755))
	fakeDatabricks := filepath.Join(fakeDir, "data'bricks")
	require.NoError(t, os.WriteFile(fakeDatabricks, []byte(`#!/bin/sh
printf '%s' "$*" > "$FAKE_ARGS_FILE"
printf '%s' "$DATABRICKS_LOG_FILE" > "$FAKE_ENV_FILE"
cat > "$FAKE_STDIN_FILE"
printf '{"Username":"oauthtoken","Secret":"secret"}\n'
`), 0o755))
	require.NoError(t, os.Chmod(fakeDatabricks, 0o755))

	shim := filepath.Join(dir, "docker-credential-databricks")
	require.NoError(t, os.WriteFile(shim, []byte(shimScript(fakeDatabricks)), 0o755))
	require.NoError(t, os.Chmod(shim, 0o755))

	cmd := exec.Command(shim, "get")
	cmd.Stdin = bytes.NewBufferString("registry-host")
	cmd.Env = append(os.Environ(),
		"DATABRICKS_LOG_FILE=stdout",
		"FAKE_ARGS_FILE="+argsPath,
		"FAKE_ENV_FILE="+envPath,
		"FAKE_STDIN_FILE="+stdinPath,
	)
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.JSONEq(t, `{"Username":"oauthtoken","Secret":"secret"}`, string(out))

	rawArgs, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	assert.Equal(t, "auth docker token", string(rawArgs))

	rawEnv, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "stderr", string(rawEnv))

	rawStdin, err := os.ReadFile(stdinPath)
	require.NoError(t, err)
	assert.Equal(t, "registry-host", string(rawStdin))

	err = exec.Command(shim, "store").Run()
	assert.Error(t, err)
	err = exec.Command(shim, "get", "store").Run()
	assert.Error(t, err)
}

func TestWindowsShimExecutesOnlyGetAndForcesLogsToStderr(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows command shim test")
	}

	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args")
	envPath := filepath.Join(dir, "env")
	stdinPath := filepath.Join(dir, "stdin")
	fakeDir := filepath.Join(dir, "bin%DOCKER_SHIM_UNSET%")
	require.NoError(t, os.Mkdir(fakeDir, 0o755))
	fakeDatabricks := filepath.Join(fakeDir, "databricks.cmd")
	require.NoError(t, os.WriteFile(fakeDatabricks, []byte(`@echo off
set /p registry=
> "%FAKE_ARGS_FILE%" echo %*
> "%FAKE_ENV_FILE%" echo %DATABRICKS_LOG_FILE%
> "%FAKE_STDIN_FILE%" echo %registry%
echo {"Username":"oauthtoken","Secret":"secret"}
`), 0o644))

	shim := filepath.Join(dir, "docker-credential-databricks.cmd")
	require.NoError(t, os.WriteFile(shim, []byte(cmdShimScript(fakeDatabricks)), 0o644))

	cmd := exec.Command(shim, "get")
	cmd.Stdin = bytes.NewBufferString("registry-host\n")
	cmd.Env = append(os.Environ(),
		"DATABRICKS_LOG_FILE=stdout",
		"FAKE_ARGS_FILE="+argsPath,
		"FAKE_ENV_FILE="+envPath,
		"FAKE_STDIN_FILE="+stdinPath,
	)
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.JSONEq(t, `{"Username":"oauthtoken","Secret":"secret"}`, string(out))

	rawArgs, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	assert.Equal(t, "auth docker token", strings.TrimSpace(string(rawArgs)))

	rawEnv, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Equal(t, "stderr", strings.TrimSpace(string(rawEnv)))

	rawStdin, err := os.ReadFile(stdinPath)
	require.NoError(t, err)
	assert.Equal(t, "registry-host", strings.TrimSpace(string(rawStdin)))

	assert.Error(t, exec.Command(shim, "store").Run())
	assert.Error(t, exec.Command(shim, "get", "store").Run())
}

func TestWindowsShimDisablesInheritedDelayedExpansion(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows command shim test")
	}

	dir := t.TempDir()
	fakeDir := filepath.Join(dir, "bin!DOCKER_SHIM_UNSET!")
	require.NoError(t, os.Mkdir(fakeDir, 0o755))
	fakeDatabricks := filepath.Join(fakeDir, "databricks.cmd")
	require.NoError(t, os.WriteFile(fakeDatabricks, []byte("@echo {\"Username\":\"oauthtoken\",\"Secret\":\"secret\"}\r\n"), 0o644))

	shim := filepath.Join(dir, "docker-credential-databricks.cmd")
	require.NoError(t, os.WriteFile(shim, []byte(cmdShimScript(fakeDatabricks)), 0o644))

	cmd := exec.Command("cmd.exe", "/D", "/V:ON", "/C", "call", shim, "get")
	cmd.Stdin = bytes.NewBufferString("registry-host\n")
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.JSONEq(t, `{"Username":"oauthtoken","Secret":"secret"}`, string(out))
}

func TestInstallShimReportsPathStatus(t *testing.T) {
	dir := t.TempDir()
	databricksPath := writeTestDatabricksExecutable(t, dir)
	t.Setenv("PATH", dir)

	got, err := InstallShim(databricksPath)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, shimFilename(runtime.GOOS)), got.Path)
	assert.True(t, got.OnPath)

	info, err := os.Stat(got.Path)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	}
}

func TestInstallShimReportsNotOnPath(t *testing.T) {
	dir := t.TempDir()
	databricksPath := writeTestDatabricksExecutable(t, dir)
	t.Setenv("PATH", t.TempDir())

	got, err := InstallShim(databricksPath)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, shimFilename(runtime.GOOS)), got.Path)
	assert.False(t, got.OnPath)
}

func TestInstallShimReportsNotOnPathWhenHelperIsShadowed(t *testing.T) {
	installDir := t.TempDir()
	databricksPath := writeTestDatabricksExecutable(t, installDir)
	shadowDir := t.TempDir()
	shadowPath := filepath.Join(shadowDir, shimFilename(runtime.GOOS))
	require.NoError(t, os.WriteFile(shadowPath, []byte("shadow"), 0o755))
	require.NoError(t, os.Chmod(shadowPath, 0o755))
	t.Setenv("PATH", shadowDir+string(os.PathListSeparator)+installDir)

	got, err := InstallShim(databricksPath)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(installDir, shimFilename(runtime.GOOS)), got.Path)
	assert.False(t, got.OnPath)
}

func TestHelperOnPathUsesDockerLookupName(t *testing.T) {
	dir := t.TempDir()
	helperPath := filepath.Join(dir, "docker-credential-databricks.cmd")
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o755))

	var gotName string
	found := helperOnPathForGOOS(helperPath, "windows", func(name string) (string, error) {
		gotName = name
		return helperPath, nil
	})

	assert.True(t, found)
	assert.Equal(t, "docker-credential-databricks", gotName)
}

func TestHelperOnPathRejectsEmptyUnixPathEntry(t *testing.T) {
	dir := t.TempDir()
	helperPath := filepath.Join(dir, shimFilename(runtime.GOOS))
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o755))

	t.Chdir(dir)
	t.Setenv("PATH", "")
	assert.False(t, helperOnPathForGOOS(helperPath, runtime.GOOS, exec.LookPath))
}

func TestSamePathUsesFileIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "docker-credential-databricks")
	link := filepath.Join(dir, "helper-link")
	require.NoError(t, os.WriteFile(target, []byte("helper"), 0o755))
	require.NoError(t, os.Symlink(target, link))

	assert.True(t, samePath(target, link))
}

func TestInstallShimDoesNotTruncateExistingHelperWhenTempCreateFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-forced failure test")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission-forced failure test requires a non-root user")
	}

	installDir := filepath.Join(t.TempDir(), "missing")
	require.NoError(t, os.MkdirAll(installDir, 0o755))
	databricksPath := writeTestDatabricksExecutable(t, installDir)
	shimPath := filepath.Join(installDir, shimFilename(runtime.GOOS))
	require.NoError(t, os.WriteFile(shimPath, []byte("existing helper"), 0o755))
	require.NoError(t, os.Chmod(installDir, 0o500))
	t.Cleanup(func() {
		_ = os.Chmod(installDir, 0o755)
	})

	_, err := InstallShim(databricksPath)
	assert.Error(t, err)

	raw, readErr := os.ReadFile(shimPath)
	require.NoError(t, readErr)
	assert.Equal(t, "existing helper", string(raw))
}
