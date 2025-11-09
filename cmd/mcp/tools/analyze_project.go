package tools

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/mcp/auth"
)

//go:embed guidance.txt
var guidanceText string

//go:embed default_readme.txt
var defaultReadmeText string

// AnalyzeProjectArgs represents the arguments for the analyze_project tool.
type AnalyzeProjectArgs struct {
	ProjectPath string `json:"project_path"`
}

// AnalyzeProject analyzes a Databricks project and returns information about it.
func AnalyzeProject(ctx context.Context, args AnalyzeProjectArgs) (string, error) {
	// Validate project path and ensure it's a Databricks project
	if err := ValidateDatabricksProject(args.ProjectPath); err != nil {
		return "", err
	}

	// Check authentication
	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
	}

	// Run bundle summary
	cmd := exec.CommandContext(ctx, GetCLIPath(), "bundle", "summary")
	cmd.Dir = args.ProjectPath

	output, err := cmd.CombinedOutput()
	var summary string
	if err != nil {
		// Include the failure output instead of erroring out
		summary = "Bundle summary failed:\n" + string(output)
	} else {
		summary = string(output)
	}

	// Get README content (heading + first paragraph)
	readmeContent := getProjectReadme(args.ProjectPath)

	// Build the result with summary, readme, and guidance
	result := fmt.Sprintf(`Project Analysis
================

%s

%s

Guidance for Working with this Project
--------------------------------------

IMPORTANT: Note that most interactions are done with the Databricks CLI; see databricks bundle --help.
IMPORTANT: To add new resources to a project, use the 'extend_project' MCP tool.

%s

Additional Resources
-------------------
- Bundle documentation: https://docs.databricks.com/dev-tools/bundles/index.html
- Bundle settings reference: https://docs.databricks.com/dev-tools/bundles/settings
- CLI reference: https://docs.databricks.com/dev-tools/cli/index.html`,
		summary, readmeContent, guidanceText)

	return result, nil
}

// GetGuidanceText returns the embedded guidance text for testing.
func GetGuidanceText() string {
	return guidanceText
}

// getReadmeHeadingAndParagraph extracts the first heading and first paragraph from a README.
// It returns the heading (with #) and the first non-empty paragraph after it.
func getReadmeHeadingAndParagraph(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	var foundHeading bool
	var foundParagraph bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for first heading
		if !foundHeading && strings.HasPrefix(trimmed, "#") {
			result = append(result, line)
			foundHeading = true
			continue
		}

		// After heading, look for first non-empty paragraph
		if foundHeading && !foundParagraph {
			if trimmed == "" {
				continue
			}
			// Found first paragraph - add it
			result = append(result, "")
			result = append(result, line)
			break
		}
	}

	if len(result) == 0 {
		return defaultReadmeText
	}

	return strings.Join(result, "\n")
}

// getProjectReadme reads the project's README.md and extracts heading + first paragraph.
// Falls back to default README if the file doesn't exist.
func getProjectReadme(projectPath string) string {
	readmePath := filepath.Join(projectPath, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		// README doesn't exist, use default
		return defaultReadmeText
	}

	return getReadmeHeadingAndParagraph(string(content))
}
