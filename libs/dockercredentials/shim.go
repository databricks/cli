package dockercredentials

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ShimInstallResult reports where the helper was installed and whether Docker can resolve it from PATH.
type ShimInstallResult struct {
	Path string
	// OnPath reports whether Path is the first matching helper in the current PATH, using PATHEXT on Windows.
	OnPath bool
}

// InstallShim installs the Docker credential helper wrapper next to the Databricks CLI.
func InstallShim(databricksPath string) (ShimInstallResult, error) {
	return installShimForGOOS(databricksPath, runtime.GOOS)
}

func installShimForGOOS(databricksPath, goos string) (ShimInstallResult, error) {
	if strings.TrimSpace(databricksPath) == "" {
		return ShimInstallResult{}, errors.New("databricks executable path is required")
	}
	installDir := filepath.Dir(databricksPath)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return ShimInstallResult{}, fmt.Errorf("create Docker credential helper directory %s: %w", installDir, err)
	}

	path := filepath.Join(installDir, shimFilename(goos))
	mode := os.FileMode(0o755)
	script := shimScript(databricksPath)
	if goos == "windows" {
		mode = 0o644
		script = cmdShimScript(databricksPath)
	}
	if err := writeShimFile(path, []byte(script), mode); err != nil {
		return ShimInstallResult{}, fmt.Errorf("write Docker credential helper %s: %w", path, err)
	}

	return ShimInstallResult{
		Path:   path,
		OnPath: helperOnPathForGOOS(path, goos, exec.LookPath),
	}, nil
}

func shimFilename(goos string) string {
	if goos == "windows" {
		return "docker-credential-" + HelperName + ".cmd"
	}
	return "docker-credential-" + HelperName
}

// shimScript accepts only get and sends CLI logs to stderr because Docker parses stdout as credential-helper JSON.
func shimScript(databricksPath string) string {
	return fmt.Sprintf(`#!/bin/sh
if [ "$#" -ne 1 ] || [ "$1" != "get" ]; then
  echo "docker-credential-databricks only supports get" >&2
  exit 1
fi
shift
export DATABRICKS_LOG_FILE=stderr
exec %s auth docker token
`, posixShellQuote(databricksPath))
}

func cmdShimScript(databricksPath string) string {
	// Percent signs must be doubled so cmd.exe treats them literally in executable paths.
	script := fmt.Sprintf(`@echo off
setlocal EnableExtensions DisableDelayedExpansion
if not "%%~1"=="get" goto unsupported
if not "%%~2"=="" goto unsupported
set DATABRICKS_LOG_FILE=stderr
"%s" auth docker token
exit /b

:unsupported
echo docker-credential-databricks only supports get 1>&2
exit /b 1
`, strings.ReplaceAll(databricksPath, "%", "%%"))
	return strings.ReplaceAll(script, "\n", "\r\n")
}

// writeShimFile preserves an existing helper if writing its replacement fails.
func writeShimFile(path string, script []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(script); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func posixShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// helperOnPathForGOOS mirrors Docker's extensionless helper lookup through PATHEXT on Windows.
func helperOnPathForGOOS(helperPath, goos string, lookPath func(string) (string, error)) bool {
	name := filepath.Base(helperPath)
	if goos == "windows" {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	candidate, err := lookPath(name)
	if err != nil {
		return false
	}
	return samePath(candidate, helperPath)
}

func samePath(a, b string) bool {
	aInfo, aErr := os.Stat(a)
	if aErr != nil {
		return false
	}
	bInfo, bErr := os.Stat(b)
	if bErr != nil {
		return false
	}
	return os.SameFile(aInfo, bInfo)
}
