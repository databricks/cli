package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ToolDefinition defines the schema for an MCP tool.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// ToolHandler is a function that executes a tool with arbitrary arguments.
type ToolHandler func(context.Context, map[string]any) (string, error)

// Tool combines a tool definition with its handler.
type Tool struct {
	Definition ToolDefinition
	Handler    ToolHandler
}

// GetAllTools returns all tools (definitions + handlers) for the MCP server.
func GetAllTools() []Tool {
	return []Tool{
		InvokeDatabricksCLITool,
		InitProjectTool,
		AnalyzeProjectTool,
		AddProjectResourceTool,
	}
}

// unmarshalArgs converts a map[string]any to a typed struct using JSON marshaling.
func unmarshalArgs(args map[string]any, target any) error {
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal arguments: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal arguments: %w", err)
	}
	return nil
}

// GetCLIPath returns the path to the current CLI executable.
// This supports development testing with ./cli.
func GetCLIPath() string {
	return os.Args[0]
}

// ValidateProjectPath checks if a project path exists and is a directory.
func ValidateProjectPath(projectPath string) error {
	if projectPath == "" {
		return errors.New("project_path is required")
	}

	pathInfo, err := os.Stat(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project directory does not exist: %s", projectPath)
		}
		return fmt.Errorf("failed to access project path: %w", err)
	}

	if !pathInfo.IsDir() {
		return fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	return nil
}

// ValidateDatabricksProject checks if a directory is a valid Databricks project.
// It ensures the directory exists and contains a databricks.yml file.
func ValidateDatabricksProject(projectPath string) error {
	if err := ValidateProjectPath(projectPath); err != nil {
		return err
	}

	databricksYml := filepath.Join(projectPath, "databricks.yml")
	if _, err := os.Stat(databricksYml); os.IsNotExist(err) {
		return fmt.Errorf("not a Databricks project: databricks.yml not found in %s\n\nUse the init_project tool to create a new project first", projectPath)
	}

	return nil
}

// cloneTemplateRepo clones a GitHub repository to a temporary directory.
// Returns the temp directory path and cleanup function.
func cloneTemplateRepo(ctx context.Context, repoURL string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", filepath.Base(repoURL)+"-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cleanup := func() { os.RemoveAll(tmpDir) }

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to clone %s: %w\nOutput: %s", filepath.Base(repoURL), err, string(output))
	}

	return tmpDir, cleanup, nil
}

// copyResourceFile copies and renames a resource YAML file from template to project.
// suffix is the file extension like ".job.yml" or ".pipeline.yml".
func copyResourceFile(resourceSrc, projectPath, resourceName, suffix string, replacements map[string]string) error {
	files, err := os.ReadDir(resourceSrc)
	if err != nil {
		return fmt.Errorf("failed to read template resources: %w", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), suffix) {
			content, err := os.ReadFile(filepath.Join(resourceSrc, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to read template: %w", err)
			}

			modified := string(content)
			for old, new := range replacements {
				modified = strings.ReplaceAll(modified, old, new)
			}

			dstFile := filepath.Join(projectPath, "resources", resourceName+suffix)
			return os.WriteFile(dstFile, []byte(modified), 0o644)
		}
	}
	return fmt.Errorf("no %s file found in template", suffix)
}

// formatTemplateList formats a list of templates as a bullet list.
func formatTemplateList(templates []string) string {
	var result strings.Builder
	for _, t := range templates {
		result.WriteString("- ")
		result.WriteString(t)
		result.WriteString("\n")
	}
	return result.String()
}

// replaceInFile replaces old string with new string in a file.
func replaceInFile(filePath string, replacements map[string]string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	modified := string(content)
	for old, new := range replacements {
		modified = strings.ReplaceAll(modified, old, new)
	}

	// Only write if content changed
	if modified != string(content) {
		return os.WriteFile(filePath, []byte(modified), 0o644)
	}
	return nil
}

// replaceInDirectory walks through a directory and replaces strings in all text files.
// It processes .py, .sql, .yml, .yaml, .md, .txt, .toml, .json, .sh files.
func replaceInDirectory(dir string, replacements map[string]string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file extension is text-based
		ext := strings.ToLower(filepath.Ext(path))
		textExts := []string{".py", ".sql", ".yml", ".yaml", ".md", ".txt", ".toml", ".json", ".sh"}
		isTextFile := false
		for _, textExt := range textExts {
			if ext == textExt {
				isTextFile = true
				break
			}
		}

		if !isTextFile {
			return nil
		}

		return replaceInFile(path, replacements)
	})
}

// contains checks if a string slice contains a specific item.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, content, info.Mode())
	})
}

// validateLanguageTemplate validates that a template is specified and is either 'python' or 'sql'.
func validateLanguageTemplate(template, resourceType string) error {
	if template == "" {
		return fmt.Errorf("template is required for %ss\n\nPlease specify the language: 'python' or 'sql'", resourceType)
	}

	if template != "python" && template != "sql" {
		return fmt.Errorf("invalid template for %s: %s. Must be 'python' or 'sql'", resourceType, template)
	}

	return nil
}
