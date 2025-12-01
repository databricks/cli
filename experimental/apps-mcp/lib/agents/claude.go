package agents

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/databricks/cli/libs/databrickscfg/profile"
)

// DetectClaude checks if Claude Code CLI is installed and available on PATH.
func DetectClaude() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// InstallClaude installs the Databricks MCP server in Claude Code.
func InstallClaude(profile *profile.Profile, warehouseID string) error {
	if !DetectClaude() {
		return errors.New("claude Code CLI is not installed or not on PATH\n\nPlease install Claude Code and ensure 'claude' is available on your system PATH.\nFor installation instructions, visit: https://docs.anthropic.com/en/docs/claude-code")
	}

	databricksPath, err := os.Executable()
	if err != nil {
		return err
	}

	removeCmd := exec.Command("claude", "mcp", "remove", "--scope", "user", "databricks-mcp")
	_ = removeCmd.Run()

	args := []string{
		"mcp", "add",
		"--scope", "user",
		"--transport", "stdio",
		"databricks-mcp",
		"--env", "DATABRICKS_CONFIG_PROFILE=" + profile.Name,
		"--env", "DATABRICKS_HOST=" + profile.Host,
		"--env", "DATABRICKS_WAREHOUSE_ID=" + warehouseID,
	}

	args = append(args, "--", databricksPath, "experimental", "apps-mcp")

	cmd := exec.Command("claude", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install MCP server in Claude Code: %w\nOutput: %s", err, string(output))
	}

	return nil
}
