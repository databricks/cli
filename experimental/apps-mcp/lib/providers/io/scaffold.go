package io

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

const (
	defaultTemplateRepo = "https://github.com/databricks/cli"
	defaultTemplateDir  = "experimental/apps-mcp/templates/appkit"
	defaultTemplateTag  = "main"
)

// ScaffoldArgs contains arguments for scaffolding operation
type ScaffoldArgs struct {
	WorkDir        string `json:"work_dir"`
	AppName        string `json:"app_name"`
	AppDescription string `json:"app_description"`
}

// ScaffoldResult contains the result of a scaffold operation
type ScaffoldResult struct {
	WorkDir      string `json:"work_dir"`
	TemplateName string `json:"template_name"`
	AppName      string `json:"app_name"`
}

// Scaffold copies template files to the work directory
func (p *Provider) Scaffold(ctx context.Context, args *ScaffoldArgs) (*ScaffoldResult, error) {
	if args.AppName == "" {
		return nil, errors.New("app name is required")
	}

	// validate that AppName only contains letters, numbers, and hyphens
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(args.AppName) {
		return nil, errors.New("app name must only contain letters, numbers, and hyphens")
	}

	normalizedAppName := strings.ToLower(args.AppName)

	if args.AppDescription == "" {
		return nil, errors.New("app description is required")
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

	// Get template data
	warehouseID, err := middlewares.GetWarehouseID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get warehouse ID: %w", err)
	}
	host := middlewares.MustGetDatabricksClient(ctx).Config.Host

	// create temp config file with parameters
	configMap := map[string]string{
		"project_name":     normalizedAppName,
		"sql_warehouse_id": warehouseID,
		"app_description":  args.AppDescription,
		"workspace_host":   host,
	}

	configBytes, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "mcp-template-config-*.json")
	if err != nil {
		return nil, fmt.Errorf("create temp config file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(configBytes); err != nil {
		return nil, fmt.Errorf("write config file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("close config file: %w", err)
	}

	// Invoke databricks CLI to initialize the bundle
	cmd := exec.CommandContext(ctx, os.Args[0], "bundle", "init",
		defaultTemplateRepo,
		"--template-dir", defaultTemplateDir,
		"--tag", defaultTemplateTag,
		"--config-file", tmpFile.Name(),
		"--output-dir", workDir,
	)
	cmd.Env = append(os.Environ(), "DATABRICKS_HOST="+host, "DATABRICKS_BUNDLE_ENGINE=direct-exp")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("databricks bundle init failed: %w\nOutput: %s", err, string(output))
	}

	log.Infof(ctx, "scaffolded project (template=%s, work_dir=%s)",
		defaultTemplateRepo+"/"+defaultTemplateDir, workDir)

	return &ScaffoldResult{
		WorkDir:      workDir,
		TemplateName: defaultTemplateRepo + "/" + defaultTemplateDir,
		AppName:      args.AppName,
	}, nil
}
