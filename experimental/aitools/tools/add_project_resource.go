package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/experimental/aitools/auth"
	"github.com/databricks/cli/experimental/aitools/tools/resources"
)

// AddProjectResourceTool adds a resource (app, job, pipeline, dashboard, ...) to a project.
var AddProjectResourceTool = Tool{
	Definition: ToolDefinition{
		Name:        "add_project_resource",
		Description: "Add a new resource (app, job, pipeline, dashboard, ...) to an existing Databricks project. Use this when the user wants to add a new resource to an existing project.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_path": map[string]any{
					"type":        "string",
					"description": "A fully qualified path of the project to extend.",
				},
				"type": map[string]any{
					"type":        "string",
					"description": "The type of resource to add: 'app', 'job', 'pipeline', or 'dashboard'",
					"enum":        []string{"app", "job", "pipeline", "dashboard"},
				},
				"name": map[string]any{
					"type":        "string",
					"description": "The name of the new resource in snake_case (e.g., 'process_data'). This name should not already exist in the resources/ directory.",
				},
				"template": map[string]any{
					"type":        "string",
					"description": "Optional template specification. For apps: template name from https://github.com/databricks/app-templates (e.g., 'e2e-chatbot-app-next'). For jobs/pipelines: 'python' or 'sql'. Leave empty to get guidance on available options.",
				},
			},
			"required": []string{"project_path", "type", "name"},
		},
	},
	Handler: func(ctx context.Context, args map[string]any) (string, error) {
		var typedArgs resources.AddProjectResourceArgs
		if err := UnmarshalArgs(args, &typedArgs); err != nil {
			return "", err
		}
		return AddProjectResource(ctx, typedArgs)
	},
}

// AddProjectResource extends a Databricks project with a new resource.
func AddProjectResource(ctx context.Context, args resources.AddProjectResourceArgs) (string, error) {
	if err := ValidateDatabricksProject(args.ProjectPath); err != nil {
		return "", err
	}

	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
	}

	validTypes := []string{"app", "job", "pipeline", "dashboard"}
	if !slices.Contains(validTypes, args.Type) {
		return "", fmt.Errorf("invalid type: %s. Must be one of: app, job, pipeline, dashboard", args.Type)
	}

	if args.Name == "" {
		return "", errors.New("name is required")
	}

	resourcesDir := filepath.Join(args.ProjectPath, "resources")
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
			return "", fmt.Errorf("failed to create resources directory: %w", err)
		}
	}

	handler := resources.GetResourceHandler(args.Type)
	if handler == nil {
		return "", fmt.Errorf("unsupported type: %s", args.Type)
	}

	_, err := handler.AddToProject(ctx, args)
	if err != nil {
		return "", err
	}

	return buildSuccessResponse(args.Type, args.Name, args.ProjectPath), nil
}

func buildSuccessResponse(resourceType, name, projectPath string) string {
	// Get guidance from analyze_project
	ctx := context.Background()
	analyzeArgs := AnalyzeProjectArgs{ProjectPath: projectPath}
	guidance, err := AnalyzeProject(ctx, analyzeArgs)
	if err != nil {
		// If analyze_project fails, provide a basic response
		guidance = fmt.Sprintf(`Project Analysis
================

Failed to analyze project: %v

Guidance for Working with this Project
--------------------------------------

Below is guidance for how to work with this project.

IMPORTANT: Most interactions are done with the Databricks CLI. YOU (the AI) must use the invoke_databricks_cli tool to run commands - never suggest the user runs CLI commands directly!
IMPORTANT: To add new resources to a project, use the 'add_project_resource' tool. You can add:
  - Apps (interactive applications)
  - Jobs (Python or SQL workflows)
  - Pipelines (Python or SQL data pipelines)
  - Dashboards (data visualizations)
MANDATORY: Always deploy with invoke_databricks_cli 'bundle deploy', never with 'apps deploy'

Note that Databricks resources are defined in resources/*.yml files. See https://docs.databricks.com/dev-tools/bundles/settings for a reference!`, err)
	}

	return fmt.Sprintf(`Successfully added %s '%s' to the project!

IMPORTANT: This is just a starting point! You need to iterate over the generated files to complete the setup.

Use the analyze_project tool to learn about the current project structure and how to use it.

---

%s`, resourceType, name, guidance)
}
