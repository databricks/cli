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

func TestShimFilename(t *testing.T) {
	require.Equal(t, "docker-credential-databricks", shimFilename("linux"))
	require.Equal(t, "docker-credential-databricks.cmd", shimFilename("windows"))
}

func TestUnixShimScript(t *testing.T) {
	got := shimScript("/opt/databricks/bin/databricks", "linux")

	require.Contains(t, got, `if [ "${1:-}" != "get" ]; then`)
	require.Contains(t, got, `docker-credential-databricks only supports get`)
	require.Contains(t, got, `export DATABRICKS_LOG_FILE=stderr`)
	require.Contains(t, got, `exec '/opt/databricks/bin/databricks' auth token --format=docker`)
}

func TestWindowsShimScript(t *testing.T) {
	got := shimScript(`C:\Program Files (x86)\Data%bricks & CLI\!DATABRICKS_LOG_FILE!\databricks.exe`, "windows")

	require.Contains(t, got, `setlocal DisableDelayedExpansion`)
	require.Contains(t, got, `if /I not "%~1"=="get"`)
	require.Contains(t, got, `docker-credential-databricks only supports get`)
	require.Contains(t, got, `shift /1`)
	require.Contains(t, got, `set "DATABRICKS_LOG_FILE=stderr"`)
	require.Contains(t, got, `"C:\Program Files (x86)\Data%%bricks ^& CLI\!DATABRICKS_LOG_FILE!\databricks.exe" auth token --format=docker`)
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
	require.NoError(t, os.WriteFile(shim, []byte(shimScript(fakeDatabricks, "linux")), 0o755))
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
}

func TestInstallShimReportsPathStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)

	got, err := InstallShim(t.Context(), "/usr/local/bin/databricks", dir)
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
	t.Setenv("PATH", t.TempDir())

	got, err := InstallShim(t.Context(), "/usr/local/bin/databricks", dir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, shimFilename(runtime.GOOS)), got.Path)
	require.False(t, got.OnPath)
}

func TestInstallShimReportsNotOnPathWhenHelperIsShadowed(t *testing.T) {
	installDir := t.TempDir()
	shadowDir := t.TempDir()
	shadowPath := filepath.Join(shadowDir, shimFilename(runtime.GOOS))
	require.NoError(t, os.WriteFile(shadowPath, []byte("shadow"), 0o755))
	require.NoError(t, os.Chmod(shadowPath, 0o755))
	t.Setenv("PATH", shadowDir+string(os.PathListSeparator)+installDir)

	got, err := InstallShim(t.Context(), "/usr/local/bin/databricks", installDir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(installDir, shimFilename(runtime.GOOS)), got.Path)
	require.False(t, got.OnPath)
}

func TestHelperOnPathUsesWindowsPathExtOrder(t *testing.T) {
	installDir := t.TempDir()
	shadowDir := t.TempDir()
	helperPath := filepath.Join(installDir, "docker-credential-databricks.cmd")
	shadowPath := filepath.Join(shadowDir, "docker-credential-databricks.exe")
	require.NoError(t, os.WriteFile(shadowPath, []byte("shadow"), 0o755))
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o644))

	env := map[string]string{
		"PATH":    shadowDir + string(os.PathListSeparator) + installDir,
		"PATHEXT": ".EXE;.CMD",
	}
	require.False(t, helperOnPathForGOOS(helperPath, "windows", func(key string) string {
		return env[key]
	}))
}

func TestHelperOnPathNormalizesWindowsPathExt(t *testing.T) {
	installDir := t.TempDir()
	helperPath := filepath.Join(installDir, "docker-credential-databricks.cmd")
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o644))

	env := map[string]string{
		"PATH":    installDir,
		"PATHEXT": "EXE;CMD",
	}
	require.True(t, helperOnPathForGOOS(helperPath, "windows", func(key string) string {
		return env[key]
	}))
}

func TestHelperOnPathIgnoresWindowsExtensionlessShadow(t *testing.T) {
	installDir := t.TempDir()
	shadowDir := t.TempDir()
	helperPath := filepath.Join(installDir, "docker-credential-databricks.cmd")
	shadowPath := filepath.Join(shadowDir, "docker-credential-databricks")
	require.NoError(t, os.WriteFile(shadowPath, []byte("shadow"), 0o755))
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o644))

	env := map[string]string{
		"PATH":    shadowDir + string(os.PathListSeparator) + installDir,
		"PATHEXT": ".COM;.EXE;.BAT;.CMD",
	}
	require.True(t, helperOnPathForGOOS(helperPath, "windows", func(key string) string {
		return env[key]
	}))
}

func TestHelperOnPathRequiresUnixExecutablePermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix mode-bit test")
	}

	installDir := t.TempDir()
	shadowDir := t.TempDir()
	helperPath := filepath.Join(installDir, "docker-credential-databricks")
	shadowPath := filepath.Join(shadowDir, "docker-credential-databricks")
	require.NoError(t, os.WriteFile(shadowPath, []byte("shadow"), 0o644))
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o755))

	env := map[string]string{
		"PATH": shadowDir + string(os.PathListSeparator) + installDir,
	}
	require.True(t, helperOnPathForGOOS(helperPath, "linux", func(key string) string {
		return env[key]
	}))
}

func TestHelperOnPathUsesUnixEffectiveUserExecutePermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix execute permission test")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can execute files that normal users cannot")
	}

	installDir := t.TempDir()
	shadowDir := t.TempDir()
	helperPath := filepath.Join(installDir, "docker-credential-databricks")
	shadowPath := filepath.Join(shadowDir, "docker-credential-databricks")
	require.NoError(t, os.WriteFile(shadowPath, []byte("shadow"), 0o001))
	require.NoError(t, os.Chmod(shadowPath, 0o001))
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o755))

	env := map[string]string{
		"PATH": shadowDir + string(os.PathListSeparator) + installDir,
	}
	require.True(t, helperOnPathForGOOS(helperPath, "linux", func(key string) string {
		return env[key]
	}))
}

func TestHelperOnPathTreatsEmptyUnixPathEntryAsWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix PATH semantics test")
	}

	dir := t.TempDir()
	helperPath := filepath.Join(dir, "docker-credential-databricks")
	require.NoError(t, os.WriteFile(helperPath, []byte("helper"), 0o755))

	oldwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(oldwd)
	})

	env := map[string]string{"PATH": ""}
	require.True(t, helperOnPathForGOOS(helperPath, "linux", func(key string) string {
		return env[key]
	}))
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

	installDir := filepath.Join(t.TempDir(), "missing")
	shimPath := filepath.Join(installDir, shimFilename(runtime.GOOS))
	require.NoError(t, os.MkdirAll(installDir, 0o755))
	require.NoError(t, os.WriteFile(shimPath, []byte("existing helper"), 0o755))
	require.NoError(t, os.Chmod(installDir, 0o500))
	t.Cleanup(func() {
		_ = os.Chmod(installDir, 0o755)
	})

	_, err := InstallShim(t.Context(), "/usr/local/bin/databricks", installDir)
	if os.Geteuid() == 0 {
		require.NoError(t, err)
		return
	}
	require.Error(t, err)

	raw, readErr := os.ReadFile(shimPath)
	require.NoError(t, readErr)
	require.Equal(t, "existing helper", string(raw))
}
