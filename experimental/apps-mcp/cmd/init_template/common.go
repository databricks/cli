package init_template

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/databricks/cli/experimental/apps-mcp/lib/common"
	"github.com/databricks/cli/experimental/apps-mcp/lib/detector"
	"github.com/databricks/cli/experimental/apps-mcp/lib/prompts"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/template"
)

// TemplateConfig holds configuration for template materialization.
type TemplateConfig struct {
	TemplatePath string // e.g., template.DefaultPython or remote URL
	TemplateName string // e.g., "default-python", "lakeflow-pipelines", "appkit"
	TemplateDir  string // subdirectory within repo (for remote templates)
	Branch       string // git branch (for remote templates)
}

// MaterializeTemplate handles the common template materialization workflow.
func MaterializeTemplate(ctx context.Context, cfg TemplateConfig, configMap map[string]any, name, outputDir string) error {
	configFile, err := writeConfigToTempFile(configMap)
	if err != nil {
		return err
	}
	defer os.Remove(configFile)

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	r := template.Resolver{
		TemplatePathOrUrl: cfg.TemplatePath,
		ConfigFile:        configFile,
		OutputDir:         outputDir,
		TemplateDir:       cfg.TemplateDir,
		Branch:            cfg.Branch,
	}

	tmpl, err := r.Resolve(ctx)
	if err != nil {
		return err
	}
	defer tmpl.Reader.Cleanup(ctx)

	if err := tmpl.Writer.Materialize(ctx, tmpl.Reader); err != nil {
		return err
	}
	tmpl.Writer.LogTelemetry(ctx)

	actualOutputDir := name
	if outputDir != "" {
		actualOutputDir = filepath.Join(outputDir, name)
	}

	absOutputDir, err := filepath.Abs(actualOutputDir)
	if err != nil {
		absOutputDir = actualOutputDir
	}

	fileCount := countFiles(absOutputDir)
	cmdio.LogString(ctx, common.FormatScaffoldSuccess(cfg.TemplateName, absOutputDir, fileCount))

	fileTree, err := generateFileTree(absOutputDir)
	if err == nil && fileTree != "" {
		cmdio.LogString(ctx, "\nFile structure:")
		cmdio.LogString(ctx, fileTree)
	}

	registry := detector.NewRegistry()
	detected := registry.Detect(ctx, absOutputDir)

	// Only write generic CLAUDE.md for non-app projects
	// (app projects have their own template-specific CLAUDE.md)
	if !detected.IsAppOnly {
		if err := writeAgentFiles(absOutputDir, map[string]any{}); err != nil {
			return fmt.Errorf("failed to write agent files: %w", err)
		}
	}

	for _, targetType := range detected.TargetTypes {
		templateName := fmt.Sprintf("target_%s.tmpl", targetType)
		if prompts.TemplateExists(templateName) {
			content := prompts.MustExecuteTemplate(templateName, map[string]any{})
			cmdio.LogString(ctx, content)
		}
	}

	return nil
}

// countFiles counts the number of files in a directory.
func countFiles(dir string) int {
	count := 0
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// writeConfigToTempFile writes a config map to a temporary JSON file.
func writeConfigToTempFile(configMap map[string]any) (string, error) {
	tmpFile, err := os.CreateTemp("", "mcp-template-config-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp config file: %w", err)
	}

	configBytes, err := json.Marshal(configMap)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("marshal config: %w", err)
	}
	if _, err := tmpFile.Write(configBytes); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("write config file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("close config file: %w", err)
	}

	return tmpFile.Name(), nil
}

// generateFileTree creates a tree-style visualization of the file structure.
// Collapses directories with more than 10 files to avoid clutter.
func generateFileTree(outputDir string) (string, error) {
	const maxFilesToShow = 10

	var allFiles []string
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(outputDir, path)
			if err != nil {
				return err
			}
			allFiles = append(allFiles, filepath.ToSlash(relPath))
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	tree := make(map[string][]string)

	for _, relPath := range allFiles {
		parts := strings.Split(relPath, "/")

		if len(parts) == 1 {
			tree[""] = append(tree[""], parts[0])
		} else {
			dir := strings.Join(parts[:len(parts)-1], "/")
			fileName := parts[len(parts)-1]
			tree[dir] = append(tree[dir], fileName)
		}
	}

	var output strings.Builder
	var sortedDirs []string
	for dir := range tree {
		sortedDirs = append(sortedDirs, dir)
	}
	sort.Strings(sortedDirs)

	for _, dir := range sortedDirs {
		filesInDir := tree[dir]
		if dir == "" {
			for _, file := range filesInDir {
				output.WriteString(file)
				output.WriteString("\n")
			}
		} else {
			output.WriteString(dir)
			output.WriteString("/\n")
			if len(filesInDir) <= maxFilesToShow {
				for _, file := range filesInDir {
					output.WriteString("  ")
					output.WriteString(file)
					output.WriteString("\n")
				}
			} else {
				output.WriteString(fmt.Sprintf("  (%d files)\n", len(filesInDir)))
			}
		}
	}

	return output.String(), nil
}

// writeAgentFiles writes CLAUDE.md and AGENTS.md files to the output directory.
func writeAgentFiles(outputDir string, data map[string]any) error {
	content := prompts.MustExecuteTemplate("AGENTS.tmpl", data)

	// Write both CLAUDE.md and AGENTS.md
	if err := os.WriteFile(filepath.Join(outputDir, "CLAUDE.md"), []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "AGENTS.md"), []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write AGENTS.md: %w", err)
	}

	return nil
}
