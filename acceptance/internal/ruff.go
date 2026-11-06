package internal

import (
	"os/exec"
	"testing"
)

// PrefetchRuff runs `uv tool run ruff@<version> --version` so that release
// lands in the uv cache before the harness sets UV_OFFLINE. Test scripts run
// ruff through the same command, offline, from that cache: not every
// environment the suite runs in has a ruff binary on PATH (the SDK bumper's
// image does not), while uv is a prerequisite anyway.
func PrefetchRuff(t *testing.T, version string) {
	out, err := exec.Command("uv", "tool", "run", "ruff@"+version, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("uv could not fetch ruff %s for the acceptance tests: %v\n%s", version, err, out)
	}
}
