package internal

import (
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/mod/semver"
)

// RequireJQ fails the run if jq is missing from PATH or older than minVersion
// (e.g. "1.7"). See the call site for why the minimum is what it is.
func RequireJQ(t *testing.T, minVersion string) {
	out, err := exec.Command("jq", "--version").Output()
	if err != nil {
		t.Fatalf("jq not found on PATH (acceptance tests require jq >= %s): %v", minVersion, err)
	}
	version := strings.TrimSpace(string(out))
	if !jqVersionOK(version, minVersion) {
		t.Fatalf("acceptance tests require jq >= %s (found %q); install a newer jq", minVersion, version)
	}
}

// jqVersionOK reports whether `jq --version` output (e.g. "jq-1.7.1") is >= minVersion.
func jqVersionOK(versionOutput, minVersion string) bool {
	got := strings.TrimPrefix(versionOutput, "jq-")
	return semver.Compare("v"+got, "v"+minVersion) >= 0
}
