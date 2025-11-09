package tools

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// InvokeDatabricksCLIArgs represents the arguments for the invoke_databricks_cli tool.
type InvokeDatabricksCLIArgs struct {
	Command          string `json:"command"`
	WorkingDirectory string `json:"working_directory,omitempty"`
}

// InvokeDatabricksCLI runs a Databricks CLI command and returns the output.
func InvokeDatabricksCLI(ctx context.Context, args InvokeDatabricksCLIArgs) (string, error) {
	// Validate command
	if args.Command == "" {
		return "", errors.New("command is required")
	}

	// Split command into arguments
	cmdArgs := strings.Fields(args.Command)

	// Create command
	cmd := exec.CommandContext(ctx, GetCLIPath(), cmdArgs...)

	// Set working directory if provided
	if args.WorkingDirectory != "" {
		cmd.Dir = args.WorkingDirectory
	}

	// Run command and capture output
	output, err := cmd.CombinedOutput()

	// Build result with stdout/stderr and exit code
	result := string(output)
	if err != nil {
		// Include exit code in error
		if exitErr, ok := err.(*exec.ExitError); ok {
			result += fmt.Sprintf("\n\nExit code: %d", exitErr.ExitCode())
		}
		return result, nil // Return output even on error
	}

	return result, nil
}
