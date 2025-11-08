package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CursorConfig represents the structure of Cursor's mcp.json file.
type CursorConfig struct {
	McpServers map[string]McpServer `json:"mcpServers"`
}

// McpServer represents an MCP server configuration in Cursor.
type McpServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// DetectCursor checks if Cursor is installed by looking for its config directory.
func DetectCursor() bool {
	configPath, err := GetCursorConfigPath()
	if err != nil {
		return false
	}
	// Check if the .cursor directory exists (not just the mcp.json file)
	cursorDir := filepath.Dir(configPath)
	return FileExists(cursorDir)
}

// InstallCursor installs the Databricks MCP server in Cursor.
func InstallCursor() error {
	configPath, err := GetCursorConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine Cursor config path: %w", err)
	}

	// Check if .cursor directory exists (not the file, we'll create that if needed)
	cursorDir := filepath.Dir(configPath)
	if !FileExists(cursorDir) {
		return fmt.Errorf("cursor directory not found at: %s\n\nPlease install Cursor from: https://cursor.sh", cursorDir)
	}

	// Read existing config
	var config CursorConfig
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist or can't be read, start with empty config
		config = CursorConfig{
			McpServers: make(map[string]McpServer),
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse Cursor config: %w", err)
		}
		if config.McpServers == nil {
			config.McpServers = make(map[string]McpServer)
		}
	}

	// Add or update the Databricks MCP server entry
	config.McpServers["databricks"] = McpServer{
		Command: "databricks",
		Args:    []string{"mcp", "server"},
	}

	// Write back to file with pretty printing
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Cursor config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0o644); err != nil {
		return fmt.Errorf("failed to write Cursor config: %w", err)
	}

	return nil
}

// NewCursorAgent creates an Agent instance for Cursor.
func NewCursorAgent() *Agent {
	detected := DetectCursor()
	return &Agent{
		Name:        "cursor",
		DisplayName: "Cursor",
		Detected:    detected,
		Installer:   InstallCursor,
	}
}
