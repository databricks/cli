package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/mcp/auth"
	"github.com/databricks/cli/experimental/mcp/tools/resources"
)

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
		if err := UnmarshalArgs(args, &typedArgs); err != nil {
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
	if readmeContent != "" {
		readmeContent = "\n\nProject-Specific Guidance\n" +
			"-------------------------\n" +
			readmeContent
	}

	resourceGuidance := getResourceGuidance(args.ProjectPath)

	const commonGuidance = `
Getting Started
---------------
Choose how you want to work on this project:

(a) Directly in your Databricks workspace, see
    https://docs.databricks.com/dev-tools/bundles/workspace.

(b) Locally with an IDE like Cursor or VS Code, see
    https://docs.databricks.com/dev-tools/vscode-ext.html.

(c) With command line tools, see https://docs.databricks.com/dev-tools/cli/databricks-cli.html

If you're developing with an IDE, dependencies for this project should be installed using uv:

*  Make sure you have the UV package manager installed.
   It's an alternative to tools like pip: https://docs.astral.sh/uv/getting-started/installation/.
*  Run ` + "`uv sync --dev`" + ` to install the project's dependencies.

Using this Project with the CLI
--------------------------------
The Databricks workspace and IDE extensions provide a graphical interface for working
with this project. It's also possible to interact with it directly using the CLI:

1. Authenticate to your Databricks workspace, if you have not done so already:
   Use invoke_databricks_cli(command="auth login --profile DEFAULT --host <workspace_host>")
   The AI needs to ask the user for the workspace host URL, it cannot guess it.

2. To deploy a development copy of this project:
   Use invoke_databricks_cli(command="bundle deploy --target dev", working_directory="<project_path>")
   (Note that "dev" is the default target, so the --target parameter is optional here.)

   This deploys everything that's defined for this project.

3. Similarly, to deploy a production copy:
   Use invoke_databricks_cli(command="bundle deploy --target prod", working_directory="<project_path>")
   Note that schedules are paused when deploying in development mode (see
   https://docs.databricks.com/dev-tools/bundles/deployment-modes.html).

4. To run a job or pipeline:
   Use invoke_databricks_cli(command="bundle run", working_directory="<project_path>")

5. To run tests locally:
   Use invoke_databricks_cli(command="bundle run <test_command>", working_directory="<project_path>")
   For Python projects, tests can be run with: ` + "`uv run pytest`" + `
`

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

%s%s%s

Additional Resources
-------------------
- Bundle documentation: https://docs.databricks.com/dev-tools/bundles/index.html
- Bundle settings reference: https://docs.databricks.com/dev-tools/bundles/settings
- CLI reference: https://docs.databricks.com/dev-tools/cli/index.html`,
		summary, commonGuidance, readmeContent, resourceGuidance)

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
// Returns empty string if the file doesn't exist (common guidance is already included).
func getProjectReadme(projectPath string) string {
	readmePath := filepath.Join(projectPath, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return ""
	}
	return skipReadmeHeadingAndParagraph(string(content))
}

// getResourceGuidance scans the resources directory and collects guidance for detected resource types.
func getResourceGuidance(projectPath string) string {
	var guidance strings.Builder
	resourcesDir := filepath.Join(projectPath, "resources")
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return ""
	}

	detected := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		var resourceType string
		switch {
		case strings.HasSuffix(name, ".app.yml") || strings.HasSuffix(name, ".app.yaml"):
			resourceType = "app"
		case strings.HasSuffix(name, ".job.yml") || strings.HasSuffix(name, ".job.yaml"):
			resourceType = "job"
		case strings.HasSuffix(name, ".pipeline.yml") || strings.HasSuffix(name, ".pipeline.yaml"):
			resourceType = "pipeline"
		case strings.HasSuffix(name, ".dashboard.yml") || strings.HasSuffix(name, ".dashboard.yaml"):
			resourceType = "dashboard"
		default:
			continue
		}
		if !detected[resourceType] {
			detected[resourceType] = true
			handler := resources.GetResourceHandler(resourceType)
			if handler != nil {
				guidanceText := handler.GetGuidancePrompt(projectPath)
				if guidanceText != "" {
					guidance.WriteString(guidanceText)
					guidance.WriteString("\n")
				}
			}
		}
	}
	return guidance.String()
}
