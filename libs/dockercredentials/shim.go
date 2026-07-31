package dockercredentials

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/env"
)

type ShimInstallResult struct {
	Path   string
	OnPath bool
}

func InstallShim(ctx context.Context, databricksPath, installDir string) (ShimInstallResult, error) {
	if strings.TrimSpace(databricksPath) == "" {
		return ShimInstallResult{}, errors.New("databricks executable path is required")
	}
	if strings.TrimSpace(installDir) == "" {
		return ShimInstallResult{}, errors.New("install directory is required")
	}

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return ShimInstallResult{}, fmt.Errorf("create Docker credential helper directory %s: %w", installDir, err)
	}

	path := filepath.Join(installDir, shimFilename(runtime.GOOS))
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		mode = 0o644
	}
	if err := writeShimFile(path, []byte(shimScript(databricksPath, runtime.GOOS)), mode); err != nil {
		return ShimInstallResult{}, fmt.Errorf("write Docker credential helper %s: %w", path, err)
	}

	return ShimInstallResult{
		Path:   path,
		OnPath: helperOnPath(ctx, path),
	}, nil
}

func shimFilename(goos string) string {
	if goos == "windows" {
		return "docker-credential-" + HelperName + ".cmd"
	}
	return "docker-credential-" + HelperName
}

func shimScript(databricksPath, goos string) string {
	if goos == "windows" {
		return fmt.Sprintf(`@echo off
setlocal DisableDelayedExpansion
if /I not "%%~1"=="get" (
  echo docker-credential-databricks only supports get 1>&2
  exit /b 1
)
set "DATABRICKS_LOG_FILE=stderr"
shift /1
%s auth token --format=docker
`, windowsBatchQuote(databricksPath))
	}

	return fmt.Sprintf(`#!/bin/sh
if [ "${1:-}" != "get" ]; then
  echo "docker-credential-databricks only supports get" >&2
  exit 1
fi
shift
export DATABRICKS_LOG_FILE=stderr
exec %s auth token --format=docker
`, posixShellQuote(databricksPath))
}

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

func windowsBatchQuote(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '%':
			b.WriteString("%%")
		case '^', '&', '|', '<', '>':
			b.WriteByte('^')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func helperOnPath(ctx context.Context, helperPath string) bool {
	return helperOnPathForGOOS(helperPath, runtime.GOOS, func(key string) string {
		return env.Get(ctx, key)
	})
}

func helperOnPathForGOOS(helperPath, goos string, getenv func(string) string) bool {
	helperName := helperLookupName(helperPath, goos)
	pathEntries := filepath.SplitList(getenv("PATH"))
	if len(pathEntries) == 0 && goos != "windows" {
		pathEntries = []string{""}
	}
	for _, entry := range pathEntries {
		if entry == "" {
			if goos == "windows" {
				continue
			}
			entry = "."
		}
		for _, candidateName := range helperLookupNames(helperName, goos, getenv("PATHEXT")) {
			candidate := filepath.Join(entry, candidateName)
			info, err := os.Stat(candidate)
			if err != nil || info.IsDir() || !isExecutableForPath(candidate, info.Mode(), goos) {
				continue
			}
			return samePath(candidate, helperPath)
		}
	}
	return false
}

func helperLookupName(helperPath, goos string) string {
	name := filepath.Base(helperPath)
	if goos == "windows" {
		return strings.TrimSuffix(name, filepath.Ext(name))
	}
	return name
}

func helperLookupNames(name, goos, pathext string) []string {
	if goos != "windows" || filepath.Ext(name) != "" {
		return []string{name}
	}
	if pathext == "" {
		pathext = ".COM;.EXE;.BAT;.CMD"
	}

	names := []string{}
	for _, ext := range strings.Split(pathext, ";") {
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		names = append(names, name+strings.ToLower(ext))
	}
	return names
}

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
