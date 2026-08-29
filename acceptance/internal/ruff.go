package internal

import (
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/mod/semver"
)

// RequireRuff fails the run if ruff is missing from PATH or older than
// minVersion (e.g. "0.9.1"). See the call site for why the minimum is what it is.
func RequireRuff(t *testing.T, minVersion string) {
	out, err := exec.Command("ruff", "--version").Output()
	if err != nil {
		t.Fatalf("ruff not found on PATH (acceptance tests require ruff >= %s): %v", minVersion, err)
	}
	version := strings.TrimSpace(string(out))
	if !ruffVersionOK(version, minVersion) {
		t.Fatalf("acceptance tests require ruff >= %s (found %q); install a newer ruff", minVersion, version)
	}
}

// ruffVersionOK reports whether `ruff --version` output (e.g. "ruff 0.9.1") is >= minVersion.
func ruffVersionOK(versionOutput, minVersion string) bool {
	fields := strings.Fields(versionOutput)
	if len(fields) < 2 {
		return false
	}
	return semver.Compare("v"+fields[1], "v"+minVersion) >= 0
}
