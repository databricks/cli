package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type cursorConfig struct {
	McpServers map[string]mcpServer `json:"mcpServers"`
}

type mcpServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func getCursorConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		userProfile := os.Getenv("USERPROFILE")
		if userProfile == "" {
			return "", os.ErrNotExist
		}
		return filepath.Join(userProfile, ".cursor", "mcp.json"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cursor", "mcp.json"), nil
}

// DetectCursor checks if Cursor is installed by looking for its config directory.
func DetectCursor() bool {
	configPath, err := getCursorConfigPath()
	if err != nil {
		return false
	}
	// Check if the .cursor directory exists (not just the mcp.json file)
	cursorDir := filepath.Dir(configPath)
	_, err = os.Stat(cursorDir)
	return err == nil
}

// InstallCursor installs the Databricks AI Tools MCP server in Cursor.
func InstallCursor() error {
	configPath, err := getCursorConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine Cursor config path: %w", err)
	}

	// Check if .cursor directory exists (not the file, we'll create that if needed)
	cursorDir := filepath.Dir(configPath)
	if _, err := os.Stat(cursorDir); err != nil {
		return fmt.Errorf("cursor directory not found at: %s\n\nPlease install Cursor from: https://cursor.sh", cursorDir)
	}

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
