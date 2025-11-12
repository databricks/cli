package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// BashArgs contains arguments for bash execution
type BashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"` // Seconds, default 120
}

// BashResult contains the result of a bash command
type BashResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// Bash executes a bash command in the workspace directory
func (p *Provider) Bash(ctx context.Context, args *BashArgs) (*BashResult, error) {
	workDir, err := p.getWorkDir()
	if err != nil {
		return nil, err
	}

	// Set timeout
	timeout := time.Duration(args.Timeout) * time.Second
	if args.Timeout == 0 {
		timeout = 120 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute command
	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Check for timeout first
	if ctx.Err() == context.DeadlineExceeded {
		timeoutSecs := args.Timeout
		if timeoutSecs == 0 {
			timeoutSecs = 120
		}
		return nil, fmt.Errorf("command timed out after %d seconds", timeoutSecs)
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return &BashResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}
