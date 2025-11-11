package resources

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AddProjectResourceArgs represents the arguments for adding a resource.
type AddProjectResourceArgs struct {
	ProjectPath string `json:"project_path"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Template    string `json:"template,omitempty"`
}

// ResourceHandler defines the interface for resource-specific operations.
type ResourceHandler interface {
	// AddToProject adds the resource to a project.
	AddToProject(ctx context.Context, args AddProjectResourceArgs) (string, error)
	// GetGuidancePrompt returns resource-specific guidance for the AI agent.
	GetGuidancePrompt(projectPath string) string
}

// CloneTemplateRepo clones a GitHub repository to a temporary directory.
func CloneTemplateRepo(ctx context.Context, repoURL string) (string, func(), error) {
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

// CopyResourceFile copies and renames a resource YAML file from template to project.
func CopyResourceFile(resourceSrc, projectPath, resourceName, suffix string, replacements map[string]string) error {
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

// FormatTemplateList formats a list of templates as a bullet list.
func FormatTemplateList(templates []string) string {
	var result strings.Builder
	for _, t := range templates {
		result.WriteString("- ")
		result.WriteString(t)
		result.WriteString("\n")
	}
	return result.String()
}

// ReplaceInDirectory walks through a directory and replaces strings in all text files.
func ReplaceInDirectory(dir string, replacements map[string]string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

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

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		modified := string(content)
		for old, new := range replacements {
			modified = strings.ReplaceAll(modified, old, new)
		}

		if modified != string(content) {
			return os.WriteFile(path, []byte(modified), 0o644)
		}
		return nil
	})
}

// CopyDir recursively copies a directory from src to dst.
func CopyDir(src, dst string) error {
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

// ValidateLanguageTemplate validates that a template is specified and is either 'python' or 'sql'.
func ValidateLanguageTemplate(template, resourceType string) error {
	if template == "" {
		return fmt.Errorf("template is required for %ss\n\nPlease specify the language: 'python' or 'sql'", resourceType)
	}

	if template != "python" && template != "sql" {
		return fmt.Errorf("invalid template for %s: %s. Must be 'python' or 'sql'", resourceType, template)
	}

	return nil
}

// GetResourceHandler returns the ResourceHandler for the given resource type.
func GetResourceHandler(resourceType string) ResourceHandler {
	switch resourceType {
	case "app":
		return &AppHandler{}
	case "job":
		return &JobHandler{}
	case "pipeline":
		return &PipelineHandler{}
	case "dashboard":
		return &DashboardHandler{}
	default:
		return nil
	}
}
