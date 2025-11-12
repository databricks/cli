package io

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/mcp/templates"
)

// ScaffoldArgs contains arguments for scaffolding operation
type ScaffoldArgs struct {
	WorkDir      string `json:"work_dir"`
	ForceRewrite bool   `json:"force_rewrite,omitempty"`
}

// ScaffoldResult contains the result of a scaffold operation
type ScaffoldResult struct {
	FilesCopied         int    `json:"files_copied"`
	WorkDir             string `json:"work_dir"`
	TemplateName        string `json:"template_name"`
	TemplateDescription string `json:"template_description"`
}

// Scaffold copies template files to the work directory
func (p *Provider) Scaffold(ctx context.Context, args *ScaffoldArgs) (*ScaffoldResult, error) {
	// Validate work directory
	workDir, err := filepath.Abs(args.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("invalid work directory: %w", err)
	}

	// Check if directory exists
	if stat, err := os.Stat(workDir); err == nil {
		if !stat.IsDir() {
			return nil, errors.New("work_dir exists but is not a directory")
		}

		// Check if empty
		entries, err := os.ReadDir(workDir)
		if err != nil {
			return nil, err
		}

		if len(entries) > 0 && !args.ForceRewrite {
			return nil, errors.New("work_dir is not empty (use force_rewrite to overwrite)")
		}

		// Clear directory if force_rewrite
		if args.ForceRewrite {
			if err := os.RemoveAll(workDir); err != nil {
				return nil, fmt.Errorf("failed to clear directory: %w", err)
			}
		}
	}

	// Create directory
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Get template
	template := p.getTemplate()
	files, err := template.Files()
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	// Copy files
	filesCopied := 0
	for path, content := range files {
		targetPath := filepath.Join(workDir, path)

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", path, err)
		}

		// Write file
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", path, err)
		}

		filesCopied++
	}

	log.Infof(ctx, "scaffolded project (template=%s, work_dir=%s, files=%d)",
		template.Name(), workDir, filesCopied)

	return &ScaffoldResult{
		FilesCopied:         filesCopied,
		WorkDir:             workDir,
		TemplateName:        template.Name(),
		TemplateDescription: template.Description(),
	}, nil
}

func (p *Provider) getTemplate() templates.Template {
	if p.config != nil && p.config.Template != nil {
		// Handle custom template if specified
		if p.config.Template.Path != "" {
			// TODO: Load from filesystem
		}
	}

	// Default to TRPC template
	return p.defaultTemplate
}
