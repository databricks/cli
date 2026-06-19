package internal

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// RequireModernRuff fails the run if ruff is missing or older than 0.9.1, the
// version pinned across the repo (python/pyproject.toml, Taskfile.yml). The
// pydabs check-formatting acceptance test runs `ruff format` and its golden
// output assumes that formatter behavior.
func RequireModernRuff(t *testing.T) {
	out, err := exec.Command("ruff", "--version").Output()
	if err != nil {
		t.Fatalf("ruff not found on PATH (acceptance tests require ruff >= 0.9.1): %v", err)
	}
	version := strings.TrimSpace(string(out))
	if !ruffVersionOK(version) {
		t.Fatalf("acceptance tests require ruff >= 0.9.1 (found %q); install a newer ruff", version)
	}
}

// ruffVersionOK reports whether `ruff --version` output (e.g. "ruff 0.9.1") is >= 0.9.1.
func ruffVersionOK(version string) bool {
	var major, minor, patch int
	if _, err := fmt.Sscanf(version, "ruff %d.%d.%d", &major, &minor, &patch); err != nil {
		return false
	}
	if major != 0 {
		return major > 0
	}
	if minor != 9 {
		return minor > 9
	}
	return patch >= 1
}
