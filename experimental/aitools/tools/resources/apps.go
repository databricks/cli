package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/aitools/tools/prompts"
)

type appHandler struct{}

func (h *appHandler) AddToProject(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command to handle all template scaffolding.

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

		templateListStr := formatTemplateList(templateList)
		errorMsg := prompts.MustExecuteTemplate("apps_pick_a_template.tmpl", map[string]string{
			"TemplateList": templateListStr,
			"ProjectPath":  args.ProjectPath,
			"Name":         args.Name,
		})
		return "", fmt.Errorf("%s", errorMsg)
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

func (h *appHandler) GetGuidancePrompt(projectPath, warehouseID, warehouseName string) string {
	return prompts.MustExecuteTemplate("apps.tmpl", map[string]string{
		"WarehouseID":   warehouseID,
		"WarehouseName": warehouseName,
	})
}

func formatTemplateList(templates []string) string {
	var result strings.Builder
	for _, t := range templates {
		result.WriteString("- ")
		result.WriteString(t)
		result.WriteString("\n")
	}
	return result.String()
}
