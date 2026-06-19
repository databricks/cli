package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// EnsureModernPython makes `python3` on PATH resolve to a Python >= 3.11, the
// version this repo's scripts target. Acceptance scripts invoke `python3`
// directly and some import stdlib modules added in 3.11 (e.g. tomllib in
// acceptance/bundle/resources/permissions/analyze_requests.py), but a host's
// default python3 may be older. uv (already required for building the
// databricks-bundles wheel) discovers or provisions a suitable interpreter; we
// symlink it as python3/python into a temp dir prepended to PATH so every
// script and build step resolves it. Fails hard if uv is missing or has no
// suitable Python.
func EnsureModernPython(t *testing.T) {
	// Windows runners already ship a python3 >= 3.11, and os.Symlink needs extra
	// privileges there, so don't provision: use the interpreter already on PATH.
	if runtime.GOOS == "windows" {
		return
	}

	out, err := exec.Command("uv", "python", "find", ">=3.11").Output()
	if err != nil {
		t.Fatalf("uv could not find python >= 3.11: %v", err)
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
