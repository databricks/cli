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
	"github.com/databricks/cli/experimental/mcp/tools/prompts"
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
					"description": "A fully qualified path of the project directory. Usually this should be in the current directory! But if it already has a lot of other things then it should be a subdirectory. Files will be created directly at this path.",
				},
			},
			"required": []string{"project_name", "project_path"},
		},
	},
	Handler: func(ctx context.Context, args map[string]any) (string, error) {
		var typedArgs initProjectArgs
		if err := UnmarshalArgs(args, &typedArgs); err != nil {
			return "", err
		}
		return InitProject(ctx, typedArgs)
	},
}

type initProjectArgs struct {
	ProjectName string `json:"project_name"`
	ProjectPath string `json:"project_path"`
}

// InitProject initializes a new Databricks project using the default-minimal template.
func InitProject(ctx context.Context, args initProjectArgs) (string, error) {
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

	// Check if a Databricks project already exists
	databricksYml := filepath.Join(args.ProjectPath, "databricks.yml")
	if _, err := os.Stat(databricksYml); err == nil {
		return "", fmt.Errorf("project already initialized: databricks.yml exists in %s\n\nUse the add_project_resource tool to add resources to this existing project", args.ProjectPath)
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
	entries, err := os.ReadDir(nestedPath)
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
	// Write instructions file (with project info included)
	instructionsContent := prompts.MustExecuteTemplate("init_project.tmpl", map[string]string{
		"ProjectName": args.ProjectName,
		"ProjectPath": args.ProjectPath,
	})
	instructionsPath := filepath.Join(args.ProjectPath, filename)
	if err := os.WriteFile(instructionsPath, []byte(instructionsContent), 0o644); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", filename, err)
	}

	// Return the same guidance as analyze_project
	result := prompts.MustExecuteTemplate("init_project.tmpl", map[string]string{
		"ProjectName": args.ProjectName,
		"ProjectPath": args.ProjectPath,
	})

	// Get project analysis and guidance
	analysis, err := AnalyzeProject(ctx, analyzeProjectArgs{ProjectPath: args.ProjectPath})
	if err != nil {
		return "", fmt.Errorf("failed to analyze initialized project: %w", err)
	}

	return result + analysis, nil
}
