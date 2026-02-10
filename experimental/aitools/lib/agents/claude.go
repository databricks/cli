package agents

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// InstallClaude installs the Databricks AI Tools MCP server in Claude Code.
func InstallClaude() error {
	// Check if claude CLI is available
	if _, err := exec.LookPath("claude"); err != nil {
		return errors.New("'claude' CLI is not installed or not on PATH\n\nPlease install Claude Code and ensure 'claude' is available on your system PATH.\nFor installation instructions, visit: https://docs.anthropic.com/en/docs/claude-code")
	}

	databricksPath, err := os.Executable()
	if err != nil {
		return err
	}

	removeCmd := exec.Command("claude", "mcp", "remove", "--scope", "user", "databricks-mcp")
	_ = removeCmd.Run()

	cmd := exec.Command("claude", "mcp", "add",
		"--scope", "user",
		"--transport", "stdio",
		"databricks-mcp",
		"--",
		databricksPath, "experimental", "aitools", "mcp")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install MCP server in Claude Code: %w\nOutput: %s", err, string(output))
	}

	return nil
}
