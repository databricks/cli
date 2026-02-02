package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type cursorConfig struct {
	McpServers map[string]mcpServer `json:"mcpServers"`
}

type mcpServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// InstallCursor installs the Databricks AI Tools MCP server in Cursor.
func InstallCursor() error {
	configDir, err := homeSubdir(".cursor")()
	if err != nil {
		return fmt.Errorf("failed to determine Cursor config path: %w", err)
	}

	// Check if .cursor directory exists
	if _, err := os.Stat(configDir); err != nil {
		return fmt.Errorf(".cursor directory not found at: %s\n\nPlease install Cursor from: https://cursor.sh", configDir)
	}

	configPath := filepath.Join(configDir, "mcp.json")

	// Read existing config
	var config cursorConfig
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist or can't be read, start with empty config
		config = cursorConfig{
			McpServers: make(map[string]mcpServer),
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse Cursor config: %w", err)
		}
		if config.McpServers == nil {
			config.McpServers = make(map[string]mcpServer)
		}
	}

	databricksPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine Databricks path: %w", err)
	}

	// Add or update the Databricks AI Tools MCP server entry
	config.McpServers["databricks-mcp"] = mcpServer{
		Command: databricksPath,
		Args:    []string{"experimental", "aitools", "mcp"},
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
