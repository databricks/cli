package validation

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// runCommand executes a shell command in the specified directory.
// Returns ValidationDetail on failure, nil on success.
func runCommand(ctx context.Context, workDir, command string) *ValidationDetail {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return &ValidationDetail{
				ExitCode: -1,
				Stdout:   stdout.String(),
				Stderr:   fmt.Sprintf("Failed to execute command: %v\nStderr: %s", err, stderr.String()),
			}
		}
	}

	if exitCode != 0 {
		return &ValidationDetail{
			ExitCode: exitCode,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
		}
	}

	return nil
}
