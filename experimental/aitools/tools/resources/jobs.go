package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/aitools/tools/prompts"
)

type jobHandler struct{}

func (h *jobHandler) AddToProject(ctx context.Context, args AddProjectResourceArgs) (string, error) {
	// FIXME: This should rely on the databricks bundle generate command to handle all template scaffolding.

	if err := ValidateLanguageTemplate(args.Template, "job"); err != nil {
		return "", err
	}

	if args.Template == "python" {
		return "", addPythonJob(ctx, args)
	}
	return "", addSQLJob(ctx, args)
}

func (h *jobHandler) GetGuidancePrompt(projectPath, warehouseID, warehouseName string) string {
	return prompts.MustLoadTemplate("jobs.tmpl")
}

func addPythonJob(ctx context.Context, args AddProjectResourceArgs) error {
	tmpDir, cleanup, err := CloneTemplateRepo(ctx, "https://github.com/databricks/bundle-examples")
	if err != nil {
		return err
	}
	defer cleanup()

	templateName := "default_python"
	templateSrc := filepath.Join(tmpDir, templateName)
	replacements := map[string]string{templateName: args.Name}

	if err := CopyResourceFile(filepath.Join(templateSrc, "resources"), args.ProjectPath, args.Name, ".job.yml", replacements); err != nil {
		return err
	}

	srcDir := filepath.Join(args.ProjectPath, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	pythonSrc := filepath.Join(templateSrc, "src", templateName)
	pythonDest := filepath.Join(srcDir, args.Name)
	if err := CopyDir(pythonSrc, pythonDest); err != nil {
		return fmt.Errorf("failed to copy python source: %w", err)
	}

	if err := ReplaceInDirectory(pythonDest, replacements); err != nil {
		return fmt.Errorf("failed to replace template references in source: %w", err)
	}

	testsDest := filepath.Join(args.ProjectPath, "tests")
	if _, err := os.Stat(testsDest); os.IsNotExist(err) {
		testsSrc := filepath.Join(templateSrc, "tests")
		if err := CopyDir(testsSrc, testsDest); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to copy tests: %v\n", err)
		} else if err := ReplaceInDirectory(testsDest, replacements); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to replace template references in tests: %v\n", err)
		}
	}
	return nil
}

func addSQLJob(ctx context.Context, args AddProjectResourceArgs) error {
	tmpDir, cleanup, err := CloneTemplateRepo(ctx, "https://github.com/databricks/bundle-examples")
	if err != nil {
		return err
	}
	defer cleanup()

	templateName := "default_sql"
	templateSrc := filepath.Join(tmpDir, templateName)
	replacements := map[string]string{templateName: args.Name}

	if err := CopyResourceFile(filepath.Join(templateSrc, "resources"), args.ProjectPath, args.Name, ".job.yml", replacements); err != nil {
		return err
	}

	srcDir := filepath.Join(args.ProjectPath, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	sqlFiles, err := filepath.Glob(filepath.Join(templateSrc, "src", "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to find SQL files: %w", err)
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
			return fmt.Errorf("failed to read SQL file: %w", err)
		}

		modified := string(content)
		for old, new := range replacements {
			modified = strings.ReplaceAll(modified, old, new)
		}

		if err := os.WriteFile(dstFile, []byte(modified), 0o644); err != nil {
			return fmt.Errorf("failed to write SQL file: %w", err)
		}
	}
	return nil
}
