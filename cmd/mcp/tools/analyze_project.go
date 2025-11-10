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

//go:embed embedded/default_python_template_readme.md
var defaultReadmeContent string

// AnalyzeProjectTool analyzes a Databricks project and returns guidance.
// It uses hardcoded guidance + guidance from the project's README.md file for this.
var AnalyzeProjectTool = Tool{
	Definition: ToolDefinition{
		Name:        "analyze_project",
		Description: "Determine what the current project is about and what actions can be performed on it. MANDATORY: Run this tool at least once per session when you see a databricks.yml file in the workspace. Also mandatory to use this for more guidance whenever the user asks things like 'run or deploy ...' or 'add ..., like add a pipeline or a job or an app' or 'change the app/dashboard/pipeline job to ...' or 'open ... in my browser' or 'preview ...'.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_path": map[string]any{
					"type":        "string",
					"description": "A fully qualified path of the project to operate on. By default, the current directory.",
				},
			},
			"required": []string{"project_path"},
		},
	},
	Handler: func(ctx context.Context, args map[string]any) (string, error) {
		var typedArgs AnalyzeProjectArgs
		if err := unmarshalArgs(args, &typedArgs); err != nil {
			return "", err
		}
		return AnalyzeProject(ctx, typedArgs)
	},
}

// AnalyzeProjectArgs represents the arguments for the analyze_project tool.
type AnalyzeProjectArgs struct {
	ProjectPath string `json:"project_path"`
}

// AnalyzeProject analyzes a Databricks project and returns information about it.
func AnalyzeProject(ctx context.Context, args AnalyzeProjectArgs) (string, error) {
	if err := ValidateDatabricksProject(args.ProjectPath); err != nil {
		return "", err
	}

	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, GetCLIPath(), "bundle", "summary")
	cmd.Dir = args.ProjectPath

	output, err := cmd.CombinedOutput()
	var summary string
	if err != nil {
		summary = "Bundle summary failed:\n" + string(output)
	} else {
		summary = string(output)
	}

	readmeContent := getProjectReadme(args.ProjectPath)

	result := fmt.Sprintf(`Project Analysis
================

%s

Guidance for Working with this Project
--------------------------------------

Below is guidance for how to work with this project.

IMPORTANT: You can suggest prompts to help users get started, such as:
  - "Create an app that shows a chart with taxi fares by city"
  - "Create a job that summarizes all taxi data using a notebook"
  - "Add a SQL pipeline to transform and aggregate customer data"
  - "Add a dashboard to visualize NYC taxi trip patterns"

IMPORTANT: Most interactions are done with the Databricks CLI. YOU (the AI) must use the invoke_databricks_cli tool to run commands - never suggest the user runs CLI commands directly!
IMPORTANT: To add new resources to a project, use the 'add_project_resource' MCP tool. You can add:
  - Apps (interactive applications)
  - Jobs (Python or SQL workflows)
  - Pipelines (Python or SQL data pipelines)
  - Dashboards (data visualizations)
MANDATORY: Always deploy with invoke_databricks_cli 'bundle deploy', never with 'apps deploy'

Note that Databricks resources are defined in resources/*.yml files. See https://docs.databricks.com/dev-tools/bundles/settings for a reference!

%s

Additional Resources
-------------------
- Bundle documentation: https://docs.databricks.com/dev-tools/bundles/index.html
- Bundle settings reference: https://docs.databricks.com/dev-tools/bundles/settings
- CLI reference: https://docs.databricks.com/dev-tools/cli/index.html`,
		summary, readmeContent)

	return result, nil
}

// skipReadmeHeadingAndParagraph removes the first heading and first paragraph, returns the rest.
func skipReadmeHeadingAndParagraph(content string) string {
	lines := strings.Split(content, "\n")
	foundHeading := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !foundHeading && strings.HasPrefix(trimmed, "#") {
			foundHeading = true
			continue
		}

		// After heading, skip empty lines then skip first paragraph
		if foundHeading && trimmed != "" {
			// Found first paragraph, return everything after this line
			if i+1 < len(lines) {
				return strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
			}
			return ""
		}
	}

	return strings.TrimSpace(content)
}

// getProjectReadme reads the project's README.md and skips heading + first paragraph.
// Falls back to default README if the file doesn't exist.
func getProjectReadme(projectPath string) string {
	readmePath := filepath.Join(projectPath, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return skipReadmeHeadingAndParagraph(defaultReadmeContent)
	}
	return skipReadmeHeadingAndParagraph(string(content))
}
