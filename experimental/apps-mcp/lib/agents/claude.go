package agents

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// DetectClaude checks if Claude Code CLI is installed and available on PATH.
func DetectClaude() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// InstallClaude installs the Databricks MCP server in Claude Code.
func InstallClaude() error {
	if !DetectClaude() {
		return errors.New("claude Code CLI is not installed or not on PATH\n\nPlease install Claude Code and ensure 'claude' is available on your system PATH.\nFor installation instructions, visit: https://docs.anthropic.com/en/docs/claude-code")
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
		databricksPath, "experimental", "apps-mcp")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install MCP server in Claude Code: %w\nOutput: %s", err, string(output))
	}

	return nil
}
