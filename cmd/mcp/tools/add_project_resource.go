package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/mcp/auth"
)

// AddProjectResourceTool adds a resource (app, job, pipeline, or dashboard) to a project.
var AddProjectResourceTool = Tool{
	Definition: ToolDefinition{
		Name:        "add_project_resource",
		Description: "Add a new resource (app, job, pipeline, or dashboard) to an existing Databricks project. Use this when the user wants to add a new resource to an existing project.",
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
		var typedArgs AddProjectResourceArgs
		if err := unmarshalArgs(args, &typedArgs); err != nil {
			return "", err
		}
		return AddProjectResource(ctx, typedArgs)
	},
}

// AddProjectResourceArgs represents the arguments for the add_project_resource tool.
type AddProjectResourceArgs struct {
	ProjectPath string `json:"project_path"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Template    string `json:"template,omitempty"`
}

// AddProjectResource extends a Databricks project with a new resource.
func AddProjectResource(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	if err := ValidateDatabricksProject(args.ProjectPath); err != nil {
		return "", err
	}

	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
	}

	validTypes := []string{"app", "job", "pipeline", "dashboard"}
	if !contains(validTypes, args.Type) {
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

	switch args.Type {
	case "app":
		return extendWithApp(ctx, args)
	case "job":
		return extendWithJob(ctx, args)
	case "pipeline":
		return extendWithPipeline(ctx, args)
	case "dashboard":
		return extendWithDashboard(ctx, args)
	default:
		return "", fmt.Errorf("unsupported type: %s", args.Type)
	}
}

func extendWithApp(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command

	tmpDir, cleanup, err := cloneTemplateRepo(ctx, "https://github.com/databricks/app-templates")
	if err != nil {
		return "", err
	}
	defer cleanup()

	if args.Template == "" {
		// List available templates
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			return "", fmt.Errorf("failed to read app-templates directory: %w", err)
		}

		var templates []string
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				templates = append(templates, entry.Name())
			}
		}

		templateList := formatTemplateList(templates)
		return "", fmt.Errorf("template parameter was not specified for the app\n\nYou have two options:\n\n1. Ask the user which template they want to use from this list:\n%s\n2. If the user described what they want the app to do, call add_project_resource again with template='nodejs-fastapi-hello-world-app' as a sensible default\n\nExample: add_project_resource(project_path='%s', type='app', name='%s', template='nodejs-fastapi-hello-world-app')",
			templateList, args.ProjectPath, args.Name)
	}

	templateSrc := filepath.Join(tmpDir, args.Template)
	if _, err := os.Stat(templateSrc); os.IsNotExist(err) {
		return "", fmt.Errorf("template '%s' not found in app-templates repository", args.Template)
	}

	appDest := filepath.Join(args.ProjectPath, args.Name)
	if err := copyDir(templateSrc, appDest); err != nil {
		return "", fmt.Errorf("failed to copy app template: %w", err)
	}

	replacements := map[string]string{
		args.Template: args.Name,
	}
	if err := replaceInDirectory(appDest, replacements); err != nil {
		return "", fmt.Errorf("failed to replace template references: %w", err)
	}

	resourceYAML := fmt.Sprintf(`resources:
  apps:
    %s:
      name: %s
      description: Databricks app created from %s template
      source_code_path: ../%s
`, args.Name, args.Name, args.Template, args.Name)

	resourceFile := filepath.Join(args.ProjectPath, "resources", args.Name+".yml")
	if err := os.WriteFile(resourceFile, []byte(resourceYAML), 0o644); err != nil {
		return "", fmt.Errorf("failed to create resource file: %w", err)
	}

	return buildSuccessResponse(args.Type, args.Name, args.ProjectPath), nil
}

func extendWithJob(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command

	if err := validateLanguageTemplate(args.Template, "job"); err != nil {
		return "", err
	}

	tmpDir, cleanup, err := cloneTemplateRepo(ctx, "https://github.com/databricks/bundle-examples")
	if err != nil {
		return "", err
	}
	defer cleanup()

	templateName := "default_" + args.Template
	templateSrc := filepath.Join(tmpDir, templateName)

	replacements := map[string]string{templateName: args.Name}
	if err := copyResourceFile(filepath.Join(templateSrc, "resources"), args.ProjectPath, args.Name, ".job.yml", replacements); err != nil {
		return "", err
	}

	srcDir := filepath.Join(args.ProjectPath, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create src directory: %w", err)
	}

	if args.Template == "python" {
		pythonSrc := filepath.Join(templateSrc, "src", templateName)
		pythonDest := filepath.Join(srcDir, args.Name)
		if err := copyDir(pythonSrc, pythonDest); err != nil {
			return "", fmt.Errorf("failed to copy python source: %w", err)
		}

		replacements := map[string]string{
			templateName: args.Name,
		}
		if err := replaceInDirectory(pythonDest, replacements); err != nil {
			return "", fmt.Errorf("failed to replace template references in source: %w", err)
		}

		testsDest := filepath.Join(args.ProjectPath, "tests")
		if _, err := os.Stat(testsDest); os.IsNotExist(err) {
			testsSrc := filepath.Join(templateSrc, "tests")
			if err := copyDir(testsSrc, testsDest); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to copy tests: %v\n", err)
			} else {
				if err := replaceInDirectory(testsDest, replacements); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to replace template references in tests: %v\n", err)
				}
			}
		}
	} else {
		sqlFiles, err := filepath.Glob(filepath.Join(templateSrc, "src", "*.sql"))
		if err != nil {
			return "", fmt.Errorf("failed to find SQL files: %w", err)
		}

		replacements := map[string]string{
			templateName: args.Name,
		}

		for _, sqlFile := range sqlFiles {
			basename := filepath.Base(sqlFile)
			newName := strings.ReplaceAll(basename, templateName, args.Name)
			dstFile := filepath.Join(srcDir, newName)

			if _, err := os.Stat(dstFile); err == nil {
				continue
			}

			content, err := os.ReadFile(sqlFile)
			if err != nil {
				return "", fmt.Errorf("failed to read SQL file: %w", err)
			}

			modified := string(content)
			for old, new := range replacements {
				modified = strings.ReplaceAll(modified, old, new)
			}

			if err := os.WriteFile(dstFile, []byte(modified), 0o644); err != nil {
				return "", fmt.Errorf("failed to write SQL file: %w", err)
			}
		}
	}

	return buildSuccessResponse(args.Type, args.Name, args.ProjectPath), nil
}

func extendWithPipeline(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command

	if err := validateLanguageTemplate(args.Template, "pipeline"); err != nil {
		return "", err
	}

	tmpDir, cleanup, err := cloneTemplateRepo(ctx, "https://github.com/databricks/bundle-examples")
	if err != nil {
		return "", err
	}
	defer cleanup()

	templateName := "lakeflow_pipelines_" + args.Template
	templateSrc := filepath.Join(tmpDir, templateName)

	// Copy source files first - use the actual directory name from bundle-examples
	srcDir := filepath.Join(templateSrc, "src")
	srcEntries, err := os.ReadDir(srcDir)
	if err != nil {
		return "", fmt.Errorf("failed to read pipeline src directory: %w", err)
	}

	// Find the first directory in src/
	var srcSubdir string
	for _, entry := range srcEntries {
		if entry.IsDir() {
			srcSubdir = entry.Name()
			break
		}
	}
	if srcSubdir == "" {
		return "", fmt.Errorf("no source directory found in %s", srcDir)
	}

	srcPattern := filepath.Join(srcDir, srcSubdir)
	srcDest := filepath.Join(args.ProjectPath, "src", args.Name)
	if err := copyDir(srcPattern, srcDest); err != nil {
		return "", fmt.Errorf("failed to copy pipeline source: %w", err)
	}

	replacements := map[string]string{
		templateName: args.Name,
		srcSubdir:    args.Name,
	}
	if err := copyResourceFile(filepath.Join(templateSrc, "resources"), args.ProjectPath, args.Name, ".pipeline.yml", replacements); err != nil {
		return "", err
	}

	if err := replaceInDirectory(srcDest, replacements); err != nil {
		return "", fmt.Errorf("failed to replace template references: %w", err)
	}

	return buildSuccessResponse(args.Type, args.Name, args.ProjectPath), nil
}

func extendWithDashboard(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command

	tmpDir, cleanup, err := cloneTemplateRepo(ctx, "https://github.com/databricks/bundle-examples")
	if err != nil {
		return "", err
	}
	defer cleanup()

	templateSrc := filepath.Join(tmpDir, "knowledge_base", "dashboard_nyc_taxi")

	// Dashboard templates use the file name (without extension) as the resource name
	resourceFiles, err := os.ReadDir(filepath.Join(templateSrc, "resources"))
	if err != nil {
		return "", fmt.Errorf("failed to read template resources: %w", err)
	}

	var oldName string
	for _, file := range resourceFiles {
		if strings.HasSuffix(file.Name(), ".dashboard.yml") {
			oldName = strings.TrimSuffix(file.Name(), ".dashboard.yml")
			break
		}
	}

	replacements := map[string]string{oldName: args.Name}
	if err := copyResourceFile(filepath.Join(templateSrc, "resources"), args.ProjectPath, args.Name, ".dashboard.yml", replacements); err != nil {
		return "", err
	}

	srcDir := filepath.Join(args.ProjectPath, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create src directory: %w", err)
	}

	dashFiles, err := filepath.Glob(filepath.Join(templateSrc, "src", "*.lvdash.json"))
	if err != nil {
		return "", fmt.Errorf("failed to find dashboard files: %w", err)
	}

	for _, dashFile := range dashFiles {
		basename := filepath.Base(dashFile)
		// Extract the old name from the filename (e.g., "nyc_taxi.lvdash.json" -> "nyc_taxi")
		oldName := strings.TrimSuffix(basename, ".lvdash.json")
		newName := args.Name + ".lvdash.json"
		dstFile := filepath.Join(srcDir, newName)

		content, err := os.ReadFile(dashFile)
		if err != nil {
			return "", fmt.Errorf("failed to read dashboard JSON: %w", err)
		}

		modified := strings.ReplaceAll(string(content), oldName, args.Name)

		if err := os.WriteFile(dstFile, []byte(modified), 0o644); err != nil {
			return "", fmt.Errorf("failed to write dashboard JSON: %w", err)
		}
	}

	return buildSuccessResponse(args.Type, args.Name, args.ProjectPath), nil
}

func buildSuccessResponse(resourceType, name, projectPath string) string {
	// Map resource types to file extensions
	extensions := map[string]string{
		"app":       "yml",
		"job":       "job.yml",
		"pipeline":  "pipeline.yml",
		"dashboard": "dashboard.yml",
	}
	ext := extensions[resourceType]
	if ext == "" {
		ext = "yml"
	}

	// Build base response
	base := fmt.Sprintf(`Successfully added %s '%s' to the project!

Files created:
- resources/%s.%s
- src/%s/

IMPORTANT: This is just a starting point! You need to iterate over the generated files to complete the setup.

Next steps:

1. Use the analyze_project tool to learn about the current project structure and how to use it

2. Validate that the extensions are correct:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="bundle validate", working_directory="%s")`, resourceType, name, name, ext, name, projectPath)

	// Add resource-specific steps
	var specificSteps string
	switch resourceType {
	case "app":
		specificSteps = fmt.Sprintf(`
   For apps: Also check that any warehouse references in resources/%s.yml are valid

3. Fix any validation errors in the configuration files

4. Test the app locally before deployment:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="apps run-local --source-dir %s", working_directory="%s")

5. Deploy to your development workspace:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="bundle deploy --target dev", working_directory="%s")`, name, name, projectPath, projectPath)
	case "pipeline":
		specificSteps = fmt.Sprintf(`
3. Fix any validation errors in the configuration files

4. Optionally, validate the pipeline definition before deployment:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="bundle run %s --validate-only", working_directory="%s")

5. Deploy to your development workspace:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="bundle deploy --target dev", working_directory="%s")`, name, projectPath, projectPath)
	default:
		specificSteps = fmt.Sprintf(`
3. Fix any validation errors in the configuration files

4. Deploy to your development workspace:
   Use invoke_databricks_cli tool: invoke_databricks_cli(command="bundle deploy --target dev", working_directory="%s")`, projectPath)
	}

	return base + specificSteps + `

For more information about bundle resources, visit:
https://docs.databricks.com/dev-tools/bundles/settings`
}
