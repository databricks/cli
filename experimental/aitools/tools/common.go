package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type clientNameKey struct{}

// SetClientName stores the MCP client name in the context.
func SetClientName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, clientNameKey{}, name)
}

// GetClientName returns the MCP client name from the context, or empty string if not available.
func GetClientName(ctx context.Context) string {
	if name, ok := ctx.Value(clientNameKey{}).(string); ok {
		return name
	}
	return ""
}

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

// GetDatabricksPath returns the path to the databricks executable.
// Returns the current executable if it appears to be a dev build, otherwise looks up "databricks" on PATH.
func GetDatabricksPath() (string, error) {
	currentExe, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Use current executable for dev builds
	if filepath.Base(currentExe) == "cli" || filepath.Base(currentExe) == "v0.0.0-dev" {
		return currentExe, nil
	}

	// Look up databricks on PATH for production usage
	path, err := exec.LookPath("databricks")
	if err != nil {
		return "", fmt.Errorf("databricks CLI not found on PATH: %w", err)
	}
	return path, nil
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
