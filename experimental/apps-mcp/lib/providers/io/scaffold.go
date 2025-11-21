package io

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/databricks/cli/experimental/apps-mcp/lib/templates"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
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

	f, err := filer.NewLocalClient(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create filer: %w", err)
	}

	// Check if directory exists
	if stat, err := f.Stat(ctx, "."); err == nil {
		if !stat.IsDir() {
			return nil, errors.New("work_dir exists but is not a directory")
		}

		// Check if empty
		entries, err := f.ReadDir(ctx, ".")
		if err != nil {
			return nil, err
		}

		if len(entries) > 0 && !args.ForceRewrite {
			return nil, errors.New("work_dir is not empty (use force_rewrite to overwrite)")
		}

		// Clear directory if force_rewrite
		if args.ForceRewrite {
			for _, entry := range entries {
				if err := f.Delete(ctx, entry.Name(), filer.DeleteRecursively); err != nil {
					return nil, fmt.Errorf("failed to delete %s: %w", entry.Name(), err)
				}
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		// Some other error
		// filer.FileDoesNotExistError implements Is(fs.ErrNotExist)
		return nil, fmt.Errorf("failed to check work directory: %w", err)
	}

	// Create directory
	if err := f.Mkdir(ctx, "."); err != nil {
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
		// filer.Write handles creating parent directories if requested
		if err := f.Write(ctx, path, bytes.NewReader([]byte(content)), filer.CreateParentDirectories); err != nil {
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
	// TODO: Support custom templates by checking p.config.Template.Path
	// and loading from filesystem. Not yet implemented.

	// Default to TRPC template
	return p.defaultTemplate
}
