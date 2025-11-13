package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/experimental/mcp/tools/prompts"
)

type pipelineHandler struct{}

func (h *pipelineHandler) AddToProject(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command to handle all template scaffolding.

	if err := ValidateLanguageTemplate(args.Template, "pipeline"); err != nil {
		return "", err
	}

	tmpDir, cleanup, err := CloneTemplateRepo(ctx, "https://github.com/databricks/bundle-examples")
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
	if err := CopyDir(srcPattern, srcDest); err != nil {
		return "", fmt.Errorf("failed to copy pipeline source: %w", err)
	}

	replacements := map[string]string{
		templateName: args.Name,
		srcSubdir:    args.Name,
	}
	if err := CopyResourceFile(filepath.Join(templateSrc, "resources"), args.ProjectPath, args.Name, ".pipeline.yml", replacements); err != nil {
		return "", err
	}

	if err := ReplaceInDirectory(srcDest, replacements); err != nil {
		return "", fmt.Errorf("failed to replace template references: %w", err)
	}

	return "", nil
}

func (h *pipelineHandler) GetGuidancePrompt(projectPath, warehouseID, warehouseName string) string {
	return prompts.MustLoadTemplate("pipelines.tmpl")
}
