package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AppHandler struct{}

func (h *AppHandler) AddToProject(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command

	tmpDir, cleanup, err := CloneTemplateRepo(ctx, "https://github.com/databricks/app-templates")
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

		var templateList []string
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				templateList = append(templateList, entry.Name())
			}
		}

		templateListStr := FormatTemplateList(templateList)
		return "", fmt.Errorf("template parameter was not specified for the app\n\nYou have two options:\n\n1. Ask the user which template they want to use from this list:\n%s\n2. If the user described what they want the app to do, call add_project_resource again with template='nodejs-fastapi-hello-world-app' as a sensible default\n\nExample: add_project_resource(project_path='%s', type='app', name='%s', template='nodejs-fastapi-hello-world-app')",
			templateListStr, args.ProjectPath, args.Name)
	}

	templateSrc := filepath.Join(tmpDir, args.Template)
	if _, err := os.Stat(templateSrc); os.IsNotExist(err) {
		return "", fmt.Errorf("template '%s' not found in app-templates repository", args.Template)
	}

	appDest := filepath.Join(args.ProjectPath, args.Name)
	if err := CopyDir(templateSrc, appDest); err != nil {
		return "", fmt.Errorf("failed to copy app template: %w", err)
	}

	replacements := map[string]string{
		args.Template: args.Name,
	}
	if err := ReplaceInDirectory(appDest, replacements); err != nil {
		return "", fmt.Errorf("failed to replace template references: %w", err)
	}

	resourceYAML := fmt.Sprintf(`resources:
  apps:
    %s:
      name: %s
      description: Databricks app created from %s template
      source_code_path: ../%s
`, args.Name, args.Name, args.Template, args.Name)

	resourceFile := filepath.Join(args.ProjectPath, "resources", args.Name+".app.yml")
	if err := os.WriteFile(resourceFile, []byte(resourceYAML), 0o644); err != nil {
		return "", fmt.Errorf("failed to create resource file: %w", err)
	}

	return "", nil
}

func (h *AppHandler) GetGuidancePrompt(projectPath string) string {
	return `
Working with Apps
----------------
- Apps are interactive applications that can be deployed to Databricks workspaces
- App source code is typically in a subdirectory matching the app name
- Before deployment, test apps locally using: invoke_databricks_cli(command="apps run-local --source-dir <app_dir>", working_directory="<project_path>")
- Validate warehouse references in resources/*.yml files are valid before deployment
- MANDATORY: Always deploy apps using: invoke_databricks_cli(command="bundle deploy --target dev", working_directory="<project_path>")
- MANDATORY: Never use 'apps deploy' directly - always use 'bundle deploy'`
}
