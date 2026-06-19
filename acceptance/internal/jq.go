package internal

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// RequireModernJq fails the run if jq is missing or older than 1.7. Acceptance
// scripts use jq 1.7 features (the pick/1 builtin and the `.foo.[]` iteration
// syntax); an older jq compiles them as errors and produces spurious diffs
// across many tests rather than one clear failure.
func RequireModernJq(t *testing.T) {
	out, err := exec.Command("jq", "--version").Output()
	if err != nil {
		t.Fatalf("jq not found on PATH (acceptance tests require jq >= 1.7): %v", err)
	}
	version := strings.TrimSpace(string(out))
	if !jqVersionOK(version) {
		t.Fatalf("acceptance tests require jq >= 1.7 (found %q); install a newer jq", version)
	}
}

// jqVersionOK reports whether `jq --version` output (e.g. "jq-1.7.1") is >= 1.7.
func jqVersionOK(version string) bool {
	var major, minor int
	if _, err := fmt.Sscanf(version, "jq-%d.%d", &major, &minor); err != nil {
		return false
	}
	return major > 1 || (major == 1 && minor >= 7)
}
