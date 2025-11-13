package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// CheckAuthentication checks if the user is authenticated to a Databricks workspace.
func CheckAuthentication(ctx context.Context) error {
	if os.Getenv("DATABRICKS_MCP_SKIP_AUTH_CHECK") == "1" {
		return nil
	}

	// Use a non-existent job ID: 404 means authenticated, 401 means not authenticated
	cliPath := os.Args[0]
	cmd := exec.CommandContext(ctx, cliPath, "jobs", "get", "999999999")
	err := cmd.Run()

	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 1 {
			return errors.New("not authenticated to Databricks\n\nTo authenticate, please run:\n  databricks auth login --profile DEFAULT --host <your-workspace-url>\n\nReplace <your-workspace-url> with your Databricks workspace URL (e.g., mycompany.cloud.databricks.com).\n\nDon't have a Databricks account? You can set up a fully free account for experimentation at:\nhttps://docs.databricks.com/getting-started/free-edition\n\nOnce authenticated, you can use this tool again")
		}
		return nil
	}

	return fmt.Errorf("failed to check authentication: %w", err)
}
