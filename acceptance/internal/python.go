package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/mod/semver"
)

// EnsurePython makes `python3` on PATH resolve to a Python >= minVersion (e.g.
// "3.11"). Acceptance scripts invoke `python3` directly and some import stdlib
// modules added in newer versions, but a host's default python3 may be older.
// On non-Windows hosts uv (see RequireUV) selects a compatible interpreter,
// which we symlink as python3/python into a temp dir prepended to PATH. On
// Windows os.Symlink needs extra privileges, so we instead require that the
// python3 already on PATH satisfies the floor.
func EnsurePython(t *testing.T, minVersion string) {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("python3", "--version").Output()
		if err != nil {
			t.Fatalf("python3 not found on PATH (acceptance tests require python >= %s): %v", minVersion, err)
		}
		version := strings.TrimSpace(string(out))
		if !pythonVersionOK(version, minVersion) {
			t.Fatalf("acceptance tests require python >= %s (found %q)", minVersion, version)
		}
		return
	}

	out, err := exec.Command("uv", "python", "find", ">="+minVersion).Output()
	if err != nil {
		t.Fatalf("uv could not find python >= %s: %v", minVersion, err)
	}
	python := strings.TrimSpace(string(out))

	binDir := t.TempDir()
	for _, link := range []string{"python3", "python"} {
		if err := os.Symlink(python, filepath.Join(binDir, link)); err != nil {
			t.Fatalf("failed to symlink %s as %s: %v", python, link, err)
		}
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH")) //nolint:forbidigo // acceptance test harness; no ctx for libs/env
	t.Logf("acceptance tests: using %s (via uv) as python3", python)
}

// pythonVersionOK reports whether `python3 --version` output (e.g. "Python 3.13.2") is >= minVersion.
func pythonVersionOK(versionOutput, minVersion string) bool {
	fields := strings.Fields(versionOutput)
	if len(fields) < 2 {
		return false
	}
	return semver.Compare("v"+fields[1], "v"+minVersion) >= 0
}
