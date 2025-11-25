package io

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/experimental/apps-mcp/lib/templates"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

// ScaffoldArgs contains arguments for scaffolding operation
type ScaffoldArgs struct {
	WorkDir        string `json:"work_dir"`
	AppName        string `json:"app_name"`
	AppDescription string `json:"app_description"`
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
	if args.AppName == "" {
		return nil, fmt.Errorf("app name is required")
	}

	// validate that AppName only contains letters, numbers, and hyphens
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(args.AppName) {
		return nil, fmt.Errorf("app name must only contain letters, numbers, and hyphens")
	}

	normalizedAppName := strings.ToLower(args.AppName)

	if args.AppDescription == "" {
		return nil, fmt.Errorf("app description is required")
	}

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

		allowedEntries := map[string]bool{
			".git":    true,
			".claude": true,
		}

		for _, entry := range entries {
			if !allowedEntries[entry.Name()] {
				return nil, fmt.Errorf("work_dir is not empty: %s", entry.Name())
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
	tmpl := p.getTemplate()
	files, err := tmpl.Files()
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	// Get template data
	warehouseID, err := middlewares.GetWarehouseID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get warehouse ID: %w", err)
	}
	host := middlewares.MustGetDatabricksClient(ctx).Config.Host

	templateData := map[string]string{
		"WarehouseID":    warehouseID,
		"WorkspaceURL":   host,
		"AppName":        normalizedAppName,
		"AppDescription": args.AppDescription,
	}

	// Copy files
	filesCopied := 0
	for path, content := range files {
		// Check if there's a corresponding .tmpl file for this path
		tmplPath := path + ".tmpl"
		if _, hasTmpl := files[tmplPath]; hasTmpl {
			// Skip this file, the .tmpl version will be processed instead
			continue
		}

		// Determine final path and content
		var finalPath string
		var finalContent string
		if strings.HasSuffix(path, ".tmpl") {
			// This is a template file, process it
			finalPath = strings.TrimSuffix(path, ".tmpl")

			// Parse and execute the template
			t, err := template.New(path).Parse(content)
			if err != nil {
				return nil, fmt.Errorf("failed to parse template %s: %w", path, err)
			}

			var buf bytes.Buffer
			if err := t.Execute(&buf, templateData); err != nil {
				return nil, fmt.Errorf("failed to execute template %s: %w", path, err)
			}
			finalContent = buf.String()
		} else {
			// Regular file, use as-is
			finalPath = path
			finalContent = content
		}

		// filer.Write handles creating parent directories if requested
		if err := f.Write(ctx, finalPath, bytes.NewReader([]byte(finalContent)), filer.CreateParentDirectories); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", finalPath, err)
		}

		filesCopied++
	}

	log.Infof(ctx, "scaffolded project (template=%s, work_dir=%s, files=%d)",
		tmpl.Name(), workDir, filesCopied)

	return &ScaffoldResult{
		FilesCopied:         filesCopied,
		WorkDir:             workDir,
		TemplateName:        tmpl.Name(),
		TemplateDescription: tmpl.Description(),
	}, nil
}

func (p *Provider) getTemplate() templates.Template {
	// TODO: Support custom templates by checking p.config.Template.Path
	// and loading from filesystem. Not yet implemented.

	// Default to TRPC template
	return p.defaultTemplate
}
