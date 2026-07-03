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

// RequirePython fails the test if no Python >= minVersion (e.g. "3.11") is
// available for the acceptance suite. It only verifies; ConfigurePython performs
// the PATH setup. On Windows os.Symlink needs extra privileges, so we require the
// python3 already on PATH to satisfy the floor; elsewhere uv must be able to find
// a compatible interpreter.
func RequirePython(t *testing.T, minVersion string) {
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

	if _, err := exec.Command("uv", "python", "find", ">="+minVersion).Output(); err != nil {
		t.Fatalf("uv could not find python >= %s: %v", minVersion, err)
	}
}

// ConfigurePython makes `python3` on PATH resolve to a Python >= minVersion for
// the rest of the run. Acceptance scripts invoke `python3` directly and some
// import stdlib modules added in newer versions, but a host's default python3
// may be older. On non-Windows hosts uv (see RequireUV) selects a compatible
// interpreter, which we symlink as python3/python into a temp dir prepended to
// PATH. On Windows we rely on the python3 already on PATH, which RequirePython
// has verified satisfies the floor.
//
// It must run on the top-level test's t: the t.Setenv and t.TempDir below are
// undone when that test returns, so calling it from a subtest would revert the
// PATH change before the rest of the suite runs.
func ConfigurePython(t *testing.T, minVersion string) {
	if runtime.GOOS == "windows" {
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
