package dockercredentials

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ShimInstallResult reports where the helper was installed and whether Docker can resolve it from PATH.
type ShimInstallResult struct {
	// Path is the filesystem path of the installed helper.
	Path string
	// OnPath reports whether Path is the first matching helper in the current PATH, using PATHEXT on Windows.
	OnPath bool
}

// InstallShim installs a Unix wrapper or a Windows CLI copy that dispatches by executable name.
func InstallShim(databricksPath, installDir string) (ShimInstallResult, error) {
	return installShimForGOOS(databricksPath, installDir, runtime.GOOS)
}

// installShimForGOOS writes a Unix wrapper or a Windows CLI copy and reports whether Docker resolves that exact helper first.
func installShimForGOOS(databricksPath, installDir, goos string) (ShimInstallResult, error) {
	if strings.TrimSpace(databricksPath) == "" {
		return ShimInstallResult{}, errors.New("databricks executable path is required")
	}
	if strings.TrimSpace(installDir) == "" {
		return ShimInstallResult{}, errors.New("install directory is required")
	}

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return ShimInstallResult{}, fmt.Errorf("create Docker credential helper directory %s: %w", installDir, err)
	}

	path := filepath.Join(installDir, shimFilename(goos))
	mode := os.FileMode(0o755)
	var err error
	if goos == "windows" {
		mode = 0o644
		err = copyShimFile(path, databricksPath, mode)
	} else {
		err = writeShimFile(path, []byte(shimScript(databricksPath)), mode)
	}
	if err != nil {
		return ShimInstallResult{}, fmt.Errorf("write Docker credential helper %s: %w", path, err)
	}

	return ShimInstallResult{
		Path:   path,
		OnPath: helperOnPathForGOOS(path, goos, exec.LookPath),
	}, nil
}

// shimFilename includes .exe on Windows so Docker can discover the helper through PATHEXT.
// See https://go.dev/src/os/exec/lp_windows.go.
func shimFilename(goos string) string {
	if goos == "windows" {
		return "docker-credential-" + HelperName + ".exe"
	}
	return "docker-credential-" + HelperName
}

// shimScript accepts only get and sends CLI logs to stderr so stdout remains valid credential-helper JSON.
// See https://docs.docker.com/reference/cli/docker/login/#credential-helper-protocol.
func shimScript(databricksPath string) string {
	return fmt.Sprintf(`#!/bin/sh
if [ "$#" -ne 1 ] || [ "$1" != "get" ]; then
  echo "docker-credential-databricks only supports get" >&2
  exit 1
fi
shift
export DATABRICKS_LOG_FILE=stderr
exec %s auth token --format=docker
`, posixShellQuote(databricksPath))
}

// writeShimFile stages the Unix wrapper beside the destination before replacing an existing helper.
func writeShimFile(path string, script []byte, mode os.FileMode) error {
	return writeShim(path, mode, func(tmp *os.File) error {
		_, err := tmp.Write(script)
		return err
	})
}

// copyShimFile stages the Windows CLI copy so a failed copy preserves an existing helper.
func copyShimFile(path, source string, mode os.FileMode) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	return writeShim(path, mode, func(tmp *os.File) error {
		_, err := io.Copy(tmp, src)
		return err
	})
}

// writeShim writes through a sibling temporary file so a failed update cannot truncate an existing helper.
func writeShim(path string, mode os.FileMode, write func(*os.File) error) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := write(tmp); err != nil {
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

// posixShellQuote returns one shell word even when the executable path contains quotes or metacharacters.
func posixShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// helperOnPathForGOOS reports whether Docker's lookup name resolves to the helper just installed.
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

// samePath compares file identity when possible and falls back to platform-normalized absolute paths.
func samePath(a, b string) bool {
	aInfo, aErr := os.Stat(a)
	bInfo, bErr := os.Stat(b)
	if aErr == nil && bErr == nil {
		return os.SameFile(aInfo, bInfo)
	}

	absA, err := filepath.Abs(a)
	if err == nil {
		a = absA
	}
	absB, err := filepath.Abs(b)
	if err == nil {
		b = absB
	}
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
