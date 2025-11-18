package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/experimental/aitools/auth"
	"github.com/databricks/cli/experimental/aitools/tools/prompts"
	"github.com/databricks/cli/experimental/aitools/tools/resources"
)

// AddProjectResourceTool adds a resource (app, job, pipeline, dashboard, ...) to a project.
var AddProjectResourceTool = Tool{
	Definition: ToolDefinition{
		Name:        "add_project_resource",
		Description: "MANDATORY - USE THIS TO ADD RESOURCES: Add a new resource (app, job, DLT pipeline, dashboard) to an existing Databricks project. Use this when the user wants to add a new resource to an existing project.",
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

	analyzeArgs := analyzeProjectArgs{ProjectPath: args.ProjectPath}
	guidance, analyzeErr := AnalyzeProject(ctx, analyzeArgs)

	data := map[string]string{
		"ResourceType": args.Type,
		"Name":         args.Name,
		"Guidance":     guidance,
	}
	if analyzeErr != nil {
		data["AnalyzeError"] = analyzeErr.Error()
	}
	return prompts.MustExecuteTemplate("add_project_resource.tmpl", data), nil
}
