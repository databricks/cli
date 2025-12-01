package clitools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/databricks/cli/experimental/apps-mcp/lib/common"
	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
)

// InvokeDatabricksCLI runs a Databricks CLI command and returns the output.
func InvokeDatabricksCLI(ctx context.Context, command []string, workingDirectory string) (string, error) {
	if len(command) == 0 {
		return "", errors.New("command is required")
	}

	workspaceClient := middlewares.MustGetDatabricksClient(ctx)
	host := workspaceClient.Config.Host

	// GetCLIPath returns the path to the current CLI executable
	cliPath := common.GetCLIPath()
	cmd := exec.CommandContext(ctx, cliPath, command...)
	cmd.Dir = workingDirectory
	env := os.Environ()
	env = append(env, "DATABRICKS_HOST="+host)
	cmd.Env = env

	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		result += fmt.Sprintf("\n\nCommand failed with error: %v", err)
		return result, nil
	}

	return result, nil
}
