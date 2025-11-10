package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ToolDefinition defines the schema for an MCP tool.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// ToolHandler is a function that executes a tool with arbitrary arguments.
type ToolHandler func(context.Context, map[string]any) (string, error)

// Tool combines a tool definition with its handler.
type Tool struct {
	Definition ToolDefinition
	Handler    ToolHandler
}

// UnmarshalArgs converts a map[string]any to a typed struct using JSON marshaling.
func UnmarshalArgs(args map[string]any, target any) error {
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal arguments: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal arguments: %w", err)
	}
	return nil
}

// GetCLIPath returns the path to the current CLI executable.
// This supports development testing with ./cli.
func GetCLIPath() string {
	return os.Args[0]
}

// ValidateDatabricksProject checks if a directory is a valid Databricks project.
// It ensures the directory exists and contains a databricks.yml file.
func ValidateDatabricksProject(projectPath string) error {
	if projectPath == "" {
		return errors.New("project_path is required")
	}

	pathInfo, err := os.Stat(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project directory does not exist: %s", projectPath)
		}
		return fmt.Errorf("failed to access project path: %w", err)
	}

	if !pathInfo.IsDir() {
		return fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	databricksYml := filepath.Join(projectPath, "databricks.yml")
	if _, err := os.Stat(databricksYml); os.IsNotExist(err) {
		return fmt.Errorf("not a Databricks project: databricks.yml not found in %s\n\nUse the init_project tool to create a new project first", projectPath)
	}

	return nil
}
