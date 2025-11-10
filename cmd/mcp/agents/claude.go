package agents

import (
	"errors"
	"fmt"
	"os/exec"
)

// DetectClaude checks if Claude Code CLI is installed and available on PATH.
func DetectClaude() bool {
	return IsOnPath("claude")
}

// InstallClaude installs the Databricks MCP server in Claude Code.
func InstallClaude() error {
	if !DetectClaude() {
		return errors.New("claude Code CLI is not installed or not on PATH\n\nPlease install Claude Code and ensure 'claude' is available on your system PATH.\nFor installation instructions, visit: https://docs.anthropic.com/en/docs/claude-code")
	}

	// First, try to remove any existing installation to make this idempotent (ignore errors if it doesn't exist)
	removeCmd := exec.Command("claude", "mcp", "remove", "--scope", "user", "databricks-cli")
	_ = removeCmd.Run()

	cmd := exec.Command("claude", "mcp", "add",
		"--scope", "user",
		"--transport", "stdio",
		"databricks-cli",              // server name
		"--",                          // separator for command
		"databricks", "mcp", "server") // command to run

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install MCP server in Claude Code: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// NewClaudeAgent creates an Agent instance for Claude Code.
func NewClaudeAgent() *Agent {
	detected := DetectClaude()
	return &Agent{
		Name:        "claude",
		DisplayName: "Claude Code",
		Detected:    detected,
		Installer:   InstallClaude,
	}
}
