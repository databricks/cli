package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CheckAuthentication checks if the user is authenticated to a Databricks workspace.
// It does this by trying to run a simple API call.
func CheckAuthentication(ctx context.Context) error {
	// Skip authentication check if running in test mode
	if os.Getenv("DATABRICKS_MCP_SKIP_AUTH_CHECK") == "1" {
		return nil
	}

	// Try to run a simple API call to check authentication
	// We use a non-existent job ID intentionally - we just want to see if we get an auth error
	// Use the current executable path to support development testing with ./cli
	cliPath := os.Args[0]
	cmd := exec.CommandContext(ctx, cliPath, "jobs", "get", "999999999")
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)

		// Check for authentication-related error messages
		if strings.Contains(outputStr, "not configured") ||
			strings.Contains(outputStr, "authentication") ||
			strings.Contains(outputStr, "credentials") ||
			strings.Contains(outputStr, "401") ||
			strings.Contains(outputStr, "Unauthorized") {
			return errors.New("not authenticated to Databricks\n\nTo authenticate, please run:\n  databricks auth login --profile DEFAULT --host <your-workspace-url>\n\nReplace <your-workspace-url> with your Databricks workspace URL (e.g., mycompany.cloud.databricks.com).\n\nDon't have a Databricks account? You can set up a fully free account for experimentation at:\nhttps://docs.databricks.com/getting-started/free-edition\n\nOnce authenticated, you can use this tool again")
		}

		// If we get a "not found" error, that's actually good - it means we're authenticated
		if strings.Contains(outputStr, "not found") || strings.Contains(outputStr, "404") || strings.Contains(outputStr, "does not exist") {
			return nil
		}

		// Some other error occurred
		return fmt.Errorf("failed to check authentication: %w\nOutput: %s", err, outputStr)
	}

	// Command succeeded (shouldn't happen with a fake job ID, but just in case)
	return nil
}
