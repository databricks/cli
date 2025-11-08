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

// InitProjectArgs represents the arguments for the init_project tool.
type InitProjectArgs struct {
	ProjectName string `json:"project_name"`
	ProjectPath string `json:"project_path"`
}

// InitProject initializes a new Databricks project using the default-minimal template.
func InitProject(ctx context.Context, args InitProjectArgs) (string, error) {
	// Validate project path
	if args.ProjectPath == "" {
		return "", errors.New("project_path is required")
	}

	// Validate project name
	if args.ProjectName == "" {
		return "", errors.New("project_name is required")
	}

	// Check if the project path exists or needs to be created
	pathInfo, err := os.Stat(args.ProjectPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the directory
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

	// Filter out .git and other hidden files for the empty check
	nonHiddenEntries := 0
	for _, entry := range entries {
		if entry.Name() != ".git" && entry.Name()[0] != '.' {
			nonHiddenEntries++
		}
	}

	if nonHiddenEntries > 0 {
		return "", fmt.Errorf("project directory is not empty: %s\nPlease use an empty directory or specify a new subdirectory", args.ProjectPath)
	}

	// Create a temporary config file for the template
	configData := map[string]string{
		"project_name":     args.ProjectName,
		"default_catalog":  "main",
		"personal_schemas": "yes",
	}

	configJSON, err := json.Marshal(configData)
	if err != nil {
		return "", fmt.Errorf("failed to create config JSON: %w", err)
	}

	// Write config to a temp file
	tmpDir := os.TempDir()
	configFile := filepath.Join(tmpDir, fmt.Sprintf("databricks-init-%s.json", args.ProjectName))
	if err := os.WriteFile(configFile, configJSON, 0o644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}
	defer os.Remove(configFile)

	// Run bundle init with default-minimal template
	// Use the current executable path to support development testing with ./cli
	cliPath := os.Args[0]
	cmd := exec.CommandContext(ctx, cliPath, "bundle", "init",
		"--config-file", configFile,
		"--output-dir", args.ProjectPath,
		"default-minimal")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to initialize project: %w\nOutput: %s", err, string(output))
	}

	// The template creates a subdirectory with the project name
	actualProjectPath := filepath.Join(args.ProjectPath, args.ProjectName)

	// Read the README.md from the created project
	readmePath := filepath.Join(actualProjectPath, "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		readmeContent = []byte("Project initialized successfully")
	}

	result := fmt.Sprintf(`Project '%s' initialized successfully at: %s

%s

Next steps:
1. Navigate to the project: cd %s
2. Review the databricks.yml configuration
3. Authenticate to Databricks: %s auth login --host <your-workspace-url>
4. Deploy to development: %s bundle deploy --target dev

To add resources to your project, use: %s bundle generate
For more information, visit: https://docs.databricks.com/dev-tools/bundles/index.html`,
		args.ProjectName, actualProjectPath, string(readmeContent), actualProjectPath, cliPath, cliPath, cliPath)

	return result, nil
}
