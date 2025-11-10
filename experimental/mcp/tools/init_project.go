package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/mcp/auth"
)

// InitProjectTool initializes a new Databricks project.
var InitProjectTool = Tool{
	Definition: ToolDefinition{
		Name:        "init_project",
		Description: "Initialize a new Databricks project structure. Use this to create a new project. After initialization, use add_project_resource to add specific resources (apps, jobs, pipelines, dashboards) as needed.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_name": map[string]any{
					"type":        "string",
					"description": "A name for this project in snake_case. Ask the user about this if it's not clear from the context.",
				},
				"project_path": map[string]any{
					"type":        "string",
					"description": "A fully qualified path of the project directory. Files will be created directly at this path, not in a subdirectory.",
				},
			},
			"required": []string{"project_name", "project_path"},
		},
	},
	Handler: func(ctx context.Context, args map[string]any) (string, error) {
		var typedArgs InitProjectArgs
		if err := UnmarshalArgs(args, &typedArgs); err != nil {
			return "", err
		}
		return InitProject(ctx, typedArgs)
	},
}

// InitProjectArgs represents the arguments for the init_project tool.
type InitProjectArgs struct {
	ProjectName string `json:"project_name"`
	ProjectPath string `json:"project_path"`
}

// InitProject initializes a new Databricks project using the default-minimal template.
func InitProject(ctx context.Context, args InitProjectArgs) (string, error) {
	if args.ProjectPath == "" {
		return "", errors.New("project_path is required")
	}

	if args.ProjectName == "" {
		return "", errors.New("project_name is required")
	}

	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
	}

	pathInfo, err := os.Stat(args.ProjectPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(args.ProjectPath, 0o755); err != nil {
				return "", fmt.Errorf("failed to create project directory: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to access project path: %w", err)
		}
	} else if !pathInfo.IsDir() {
		return "", fmt.Errorf("project path exists but is not a directory: %s", args.ProjectPath)
	}

	// Check if directory is empty (or nearly empty, allowing .git)
	entries, err := os.ReadDir(args.ProjectPath)
	if err != nil {
		return "", fmt.Errorf("failed to read project directory: %w", err)
	}

	var nonHiddenFiles []string
	for _, entry := range entries {
		if entry.Name() != ".git" && entry.Name()[0] != '.' {
			nonHiddenFiles = append(nonHiddenFiles, entry.Name())
		}
	}

	if len(nonHiddenFiles) > 0 {
		return "", fmt.Errorf("project directory is not empty: %s\n\nFound files/directories: %v\n\nPlease either:\n1. Use an empty directory, or\n2. Specify a new subdirectory path that doesn't exist yet", args.ProjectPath, nonHiddenFiles)
	}

	configData := map[string]string{
		"project_name":     args.ProjectName,
		"default_catalog":  "main",
		"personal_schemas": "yes",
	}

	configJSON, err := json.Marshal(configData)
	if err != nil {
		return "", fmt.Errorf("failed to create config JSON: %w", err)
	}

	tmpDir := os.TempDir()
	configFile := filepath.Join(tmpDir, fmt.Sprintf("databricks-init-%s.json", args.ProjectName))
	if err := os.WriteFile(configFile, configJSON, 0o644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}
	defer os.Remove(configFile)

	cmd := exec.CommandContext(ctx, GetCLIPath(), "bundle", "init",
		"--config-file", configFile,
		"--output-dir", args.ProjectPath,
		"default-minimal")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to initialize project: %w\nOutput: %s", err, string(output))
	}

	// The template creates a subdirectory with the project name
	// We need to move everything up one level to args.ProjectPath
	nestedPath := filepath.Join(args.ProjectPath, args.ProjectName)

	// Move all contents from nestedPath to args.ProjectPath
	entries, err = os.ReadDir(nestedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read nested project directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(nestedPath, entry.Name())
		dstPath := filepath.Join(args.ProjectPath, entry.Name())

		if err := os.Rename(srcPath, dstPath); err != nil {
			return "", fmt.Errorf("failed to move %s to project root: %w", entry.Name(), err)
		}
	}

	if err := os.Remove(nestedPath); err != nil {
		return "", fmt.Errorf("failed to remove nested directory: %w", err)
	}

	// Create agent instructions file based on the calling client
	filename := "AGENTS.md"
	clientName := GetClientName(ctx)
	if strings.Contains(strings.ToLower(clientName), "claude") {
		filename = "CLAUDE.md"
	}
	instructionsContent := `# Agent instructions

This file can be used for any project-specific instructions!

## Prerequisites

If the Databricks CLI MCP server is not yet installed, install it by:
1. Installing the Databricks CLI: https://docs.databricks.com/dev-tools/cli/install
2. Running: ` + "`databricks mcp install`" + `

## Working with this project

General agent guidance: always use the mcp__databricks-cli__analyze_project tool whenever you open this project!
It makes sure you have more context on the current project and what actions you can perform on it.
`
	instructionsPath := filepath.Join(args.ProjectPath, filename)
	if err := os.WriteFile(instructionsPath, []byte(instructionsContent), 0o644); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", filename, err)
	}

	// Return the same guidance as analyze_project
	result := fmt.Sprintf(`Project '%s' initialized successfully at: %s

⚠️  IMPORTANT: This is an EMPTY project with NO resources (no apps, jobs, pipelines, or dashboards)!

If the user asked you to create a specific resource (like "create an app" or "create a job"), you MUST now call the add_project_resource tool to add it!

Example: add_project_resource(project_path="%s", type="app", name="my_app", template="nodejs-fastapi-hello-world-app")

`, args.ProjectName, args.ProjectPath, args.ProjectPath)

	// Get project analysis and guidance
	analysis, err := AnalyzeProject(ctx, AnalyzeProjectArgs{ProjectPath: args.ProjectPath})
	if err != nil {
		return "", fmt.Errorf("failed to analyze initialized project: %w", err)
	}

	return result + analysis, nil
}
