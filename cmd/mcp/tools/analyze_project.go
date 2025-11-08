package tools

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/cmd/mcp/auth"
)

//go:embed guidance.txt
var guidanceText string

// AnalyzeProjectArgs represents the arguments for the analyze_project tool.
type AnalyzeProjectArgs struct {
	ProjectPath string `json:"project_path"`
}

// AnalyzeProject analyzes a Databricks project and returns information about it.
func AnalyzeProject(ctx context.Context, args AnalyzeProjectArgs) (string, error) {
	// Validate project path
	if args.ProjectPath == "" {
		return "", errors.New("project_path is required")
	}

	// Check if the project path exists
	pathInfo, err := os.Stat(args.ProjectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("project directory does not exist: %s", args.ProjectPath)
		}
		return "", fmt.Errorf("failed to access project path: %w", err)
	}

	if !pathInfo.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", args.ProjectPath)
	}

	// Check if databricks.yml exists
	databricksYml := filepath.Join(args.ProjectPath, "databricks.yml")
	if _, err := os.Stat(databricksYml); os.IsNotExist(err) {
		return "", fmt.Errorf("not a Databricks project: databricks.yml not found in %s\n\nUse the init_project tool to create a new project first", args.ProjectPath)
	}

	// Check authentication
	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
	}

	// Run bundle summary
	// Use the current executable path to support development testing with ./cli
	cliPath := os.Args[0]
	cmd := exec.CommandContext(ctx, cliPath, "bundle", "summary")
	cmd.Dir = args.ProjectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run bundle summary: %w\nOutput: %s", err, string(output))
	}

	summary := string(output)

	// Build the result with summary and guidance
	result := fmt.Sprintf(`Project Analysis
================

%s

%s

Additional Resources
-------------------
- Bundle documentation: https://docs.databricks.com/dev-tools/bundles/index.html
- Bundle settings reference: https://docs.databricks.com/dev-tools/bundles/settings
- CLI reference: https://docs.databricks.com/dev-tools/cli/index.html`,
		summary, guidanceText)

	return result, nil
}

// GetGuidanceText returns the embedded guidance text for testing.
func GetGuidanceText() string {
	return guidanceText
}
