package agents

import (
	"fmt"
	"os/exec"
)

// LookPath resolves a binary on PATH. It is a package-level var so tests can
// inject a fake resolver without depending on the host PATH. It mirrors the
// exec.LookPath precedent in libs/python/detect.go.
var LookPath = exec.LookPath

// ResolveBinary resolves the agent's CLI binary to an absolute path. It refuses
// a binary that resolves only relative to the current directory (exec.ErrDot),
// so a malicious ./claude is never executed.
func ResolveBinary(agent *Agent) (string, error) {
	if agent.Binary == "" {
		return "", fmt.Errorf("%s has no CLI binary", agent.DisplayName)
	}
	path, err := LookPath(agent.Binary)
	if err != nil {
		return "", fmt.Errorf("could not resolve %s on PATH: %w", agent.Binary, err)
	}
	return path, nil
}
