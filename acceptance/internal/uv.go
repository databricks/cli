package internal

import (
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/mod/semver"
)

// RequireUV fails the run if uv is missing from PATH or older than minVersion
// (e.g. "0.4"). See the call site for why the minimum is what it is.
func RequireUV(t *testing.T, minVersion string) {
	out, err := exec.Command("uv", "--version").Output()
	if err != nil {
		t.Fatalf("uv not found on PATH (acceptance tests require uv >= %s): %v", minVersion, err)
	}
	version := strings.TrimSpace(string(out))
	if !uvVersionOK(version, minVersion) {
		t.Fatalf("acceptance tests require uv >= %s (found %q); install a newer uv", minVersion, version)
	}
}

// uvVersionOK reports whether `uv --version` output (e.g. "uv 0.11.22 (abc 2025-01-01)") is >= minVersion.
func uvVersionOK(versionOutput, minVersion string) bool {
	fields := strings.Fields(versionOutput)
	if len(fields) < 2 {
		return false
	}
	return semver.Compare("v"+fields[1], "v"+minVersion) >= 0
}
