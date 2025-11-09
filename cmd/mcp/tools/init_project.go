package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/cmd/mcp/auth"
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

	// Check authentication
	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
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
	var nonHiddenFiles []string
	for _, entry := range entries {
		if entry.Name() != ".git" && entry.Name()[0] != '.' {
			nonHiddenFiles = append(nonHiddenFiles, entry.Name())
		}
	}

	if len(nonHiddenFiles) > 0 {
		return "", fmt.Errorf("project directory is not empty: %s\n\nFound files/directories: %v\n\nPlease either:\n1. Use an empty directory, or\n2. Specify a new subdirectory path that doesn't exist yet", args.ProjectPath, nonHiddenFiles)
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

	// Remove the now-empty nested directory
	if err := os.Remove(nestedPath); err != nil {
		return "", fmt.Errorf("failed to remove nested directory: %w", err)
	}

	// Read the README.md from the project root
	readmePath := filepath.Join(args.ProjectPath, "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		readmeContent = []byte("Project initialized successfully")
	}

	cliPath := GetCLIPath()
	result := fmt.Sprintf(`Project '%s' initialized successfully at: %s

%s

Next steps:
1. Navigate to the project: cd %s
2. Review the databricks.yml configuration
3. Authenticate to Databricks: %s auth login --host <your-workspace-url>
4. Deploy to development: %s bundle deploy --target dev

To add resources to your project, use: %s bundle generate
For more information, visit: https://docs.databricks.com/dev-tools/bundles/index.html`,
		args.ProjectName, args.ProjectPath, string(readmeContent), args.ProjectPath, cliPath, cliPath, cliPath)

	return result, nil
}
