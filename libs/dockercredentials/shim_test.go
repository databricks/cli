package dockercredentials

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeTestDatabricksExecutable uses the platform suffix so tests exercise Windows helper paths.
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
	require.Equal(t, "docker-credential-databricks", shimFilename("linux"))
	require.Equal(t, "docker-credential-databricks.exe", shimFilename("windows"))
}

func TestUnixShimScript(t *testing.T) {
	got := shimScript("/opt/databricks/bin/databricks")

	require.Contains(t, got, `if [ "$#" -ne 1 ] || [ "$1" != "get" ]; then`)
	require.Contains(t, got, `docker-credential-databricks only supports get`)
	require.Contains(t, got, `export DATABRICKS_LOG_FILE=stderr`)
	require.Contains(t, got, `exec '/opt/databricks/bin/databricks' auth token --format=docker`)
}

func TestInstallWindowsShimCopiesDatabricksExecutable(t *testing.T) {
	dir := t.TempDir()
	databricksPath := filepath.Join(dir, "databricks.exe")
	require.NoError(t, os.WriteFile(databricksPath, []byte("databricks executable"), 0o755))
	installDir := filepath.Join(dir, "bin")
	t.Setenv("PATH", installDir)

	got, err := installShimForGOOS(databricksPath, installDir, "windows")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(installDir, "docker-credential-databricks.exe"), got.Path)

	raw, err := os.ReadFile(got.Path)
	require.NoError(t, err)
	require.Equal(t, "databricks executable", string(raw))
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
	require.JSONEq(t, `{"Username":"oauthtoken","Secret":"secret"}`, string(out))

	rawArgs, err := os.ReadFile(argsPath)
	require.NoError(t, err)
	require.Equal(t, "auth token --format=docker", string(rawArgs))

	rawEnv, err := os.ReadFile(envPath)
	require.NoError(t, err)
	require.Equal(t, "stderr", string(rawEnv))

	rawStdin, err := os.ReadFile(stdinPath)
	require.NoError(t, err)
	require.Equal(t, "registry-host", string(rawStdin))

	err = exec.Command(shim, "store").Run()
	require.Error(t, err)
	err = exec.Command(shim, "get", "store").Run()
	require.Error(t, err)
}

func TestInstallShimReportsPathStatus(t *testing.T) {
	dir := t.TempDir()
	databricksPath := writeTestDatabricksExecutable(t, t.TempDir())
	t.Setenv("PATH", dir)

	got, err := InstallShim(databricksPath, dir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, shimFilename(runtime.GOOS)), got.Path)
	require.True(t, got.OnPath)

	info, err := os.Stat(got.Path)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	}
}

func TestInstallShimReportsNotOnPath(t *testing.T) {
	dir := t.TempDir()
	databricksPath := writeTestDatabricksExecutable(t, t.TempDir())
	t.Setenv("PATH", t.TempDir())

	got, err := InstallShim(databricksPath, dir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, shimFilename(runtime.GOOS)), got.Path)
	require.False(t, got.OnPath)
}

func TestInstallShimReportsNotOnPathWhenHelperIsShadowed(t *testing.T) {
	installDir := t.TempDir()
	databricksPath := writeTestDatabricksExecutable(t, t.TempDir())
	shadowDir := t.TempDir()
	shadowPath := filepath.Join(shadowDir, shimFilename(runtime.GOOS))
	require.NoError(t, os.WriteFile(shadowPath, []byte("shadow"), 0o755))
	require.NoError(t, os.Chmod(shadowPath, 0o755))
	t.Setenv("PATH", shadowDir+string(os.PathListSeparator)+installDir)

	got, err := InstallShim(databricksPath, installDir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(installDir, shimFilename(runtime.GOOS)), got.Path)
	require.False(t, got.OnPath)
}

func TestHelperOnPathUsesDockerLookupName(t *testing.T) {
	dir := t.TempDir()
	helperPath := filepath.Join(dir, "docker-credential-databricks.exe")
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o755))

	var gotName string
	found := helperOnPathForGOOS(helperPath, "windows", func(name string) (string, error) {
		gotName = name
		return helperPath, nil
	})

	require.True(t, found)
	require.Equal(t, "docker-credential-databricks", gotName)
}

func TestHelperOnPathRejectsEmptyUnixPathEntry(t *testing.T) {
	dir := t.TempDir()
	helperPath := filepath.Join(dir, shimFilename(runtime.GOOS))
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o755))

	t.Chdir(dir)
	t.Setenv("PATH", "")
	require.False(t, helperOnPathForGOOS(helperPath, runtime.GOOS, exec.LookPath))
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

	require.True(t, samePath(target, link))
}

func TestInstallShimDoesNotTruncateExistingHelperWhenTempCreateFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-forced failure test")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission-forced failure test requires a non-root user")
	}

	installDir := filepath.Join(t.TempDir(), "missing")
	databricksPath := writeTestDatabricksExecutable(t, t.TempDir())
	shimPath := filepath.Join(installDir, shimFilename(runtime.GOOS))
	require.NoError(t, os.MkdirAll(installDir, 0o755))
	require.NoError(t, os.WriteFile(shimPath, []byte("existing helper"), 0o755))
	require.NoError(t, os.Chmod(installDir, 0o500))
	t.Cleanup(func() {
		_ = os.Chmod(installDir, 0o755)
	})

	_, err := InstallShim(databricksPath, installDir)
	require.Error(t, err)

	raw, readErr := os.ReadFile(shimPath)
	require.NoError(t, readErr)
	require.Equal(t, "existing helper", string(raw))
}
