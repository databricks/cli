package agents

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

	databricksPath, err := getDatabricksPath()
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
		databricksPath, "mcp", "server")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install MCP server in Claude Code: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// getDatabricksPath returns the path to the databricks executable.
func getDatabricksPath() (string, error) {
	currentExe, err := os.Executable()
	if err != nil {
		return "", err
	}

	if strings.Contains(currentExe, "v0.0.0-dev") || strings.HasSuffix(currentExe, "/cli") {
		return currentExe, nil
	}

	return exec.LookPath("databricks")
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
