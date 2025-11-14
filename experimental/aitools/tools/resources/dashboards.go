package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/aitools/tools/prompts"
)

type dashboardHandler struct{}

func (h *dashboardHandler) AddToProject(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command to handle all template scaffolding.

	tmpDir, cleanup, err := CloneTemplateRepo(ctx, "https://github.com/databricks/bundle-examples")
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
	if err := CopyResourceFile(filepath.Join(templateSrc, "resources"), args.ProjectPath, args.Name, ".dashboard.yml", replacements); err != nil {
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

	return "", nil
}

func (h *dashboardHandler) GetGuidancePrompt(projectPath, warehouseID, warehouseName string) string {
	return prompts.MustLoadTemplate("dashboards.tmpl")
}
