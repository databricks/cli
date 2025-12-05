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
func InvokeDatabricksCLI(ctx context.Context, args []string, workingDirectory string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("args is required")
	}

	workspaceClient, err := middlewares.GetDatabricksClient(ctx)
	if err != nil {
		return "", fmt.Errorf("get databricks client: %w", err)
	}
	host := workspaceClient.Config.Host
	profile := middlewares.GetDatabricksProfile(ctx)

	// GetCLIPath returns the path to the current CLI executable
	cliPath := common.GetCLIPath()
	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Dir = workingDirectory
	env := os.Environ()
	env = append(env, "DATABRICKS_HOST="+host)
	if profile != "" {
		env = append(env, "DATABRICKS_CONFIG_PROFILE="+profile)
	}
	cmd.Env = env

	output, err := cmd.CombinedOutput()

	result := string(output)
	if err != nil {
		result += fmt.Sprintf("\n\nCommand failed with error: %v", err)
		return result, nil
	}

	return result, nil
}
