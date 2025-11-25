package clitools

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/experimental/apps-mcp/lib/common"
	"github.com/databricks/cli/libs/exec"
)

// InvokeDatabricksCLI runs a Databricks CLI command and returns the output.
func InvokeDatabricksCLI(ctx context.Context, command string, workingDirectory *string) (string, error) {
	if command == "" {
		return "", errors.New("command is required")
	}

	workDir := "."
	if workingDirectory != nil && *workingDirectory != "" {
		workDir = *workingDirectory
	}

	executor, err := exec.NewCommandExecutor(workDir)
	if err != nil {
		return "", fmt.Errorf("failed to create command executor: %w", err)
	}

	// GetCLIPath returns the path to the current CLI executable
	cliPath := common.GetCLIPath()
	fullCommand := fmt.Sprintf(`"%s" %s`, cliPath, command)
	output, err := executor.Exec(ctx, fullCommand)

	result := string(output)
	if err != nil {
		result += fmt.Sprintf("\n\nCommand failed with error: %v", err)
		return result, nil
	}

	return result, nil
}
