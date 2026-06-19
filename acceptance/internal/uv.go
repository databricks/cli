package internal

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// RequireModernUv fails the run if uv is missing or older than 0.4. uv builds
// the databricks-bundles wheel and provides the test interpreter via
// `uv python find` (see EnsureModernPython), which landed in the 0.3 line; 0.4
// is a small margin above that.
func RequireModernUv(t *testing.T) {
	out, err := exec.Command("uv", "--version").Output()
	if err != nil {
		t.Fatalf("uv not found on PATH (acceptance tests require uv >= 0.4): %v", err)
	}
	version := strings.TrimSpace(string(out))
	if !uvVersionOK(version) {
		t.Fatalf("acceptance tests require uv >= 0.4 (found %q); install a newer uv", version)
	}
}

// uvVersionOK reports whether `uv --version` output (e.g. "uv 0.11.22 (abc)") is >= 0.4.
func uvVersionOK(version string) bool {
	var major, minor int
	if _, err := fmt.Sscanf(version, "uv %d.%d", &major, &minor); err != nil {
		return false
	}
	return major > 0 || minor >= 4
}
