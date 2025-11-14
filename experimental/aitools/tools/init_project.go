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

	"github.com/databricks/cli/experimental/aitools/auth"
	"github.com/databricks/cli/experimental/aitools/tools/prompts"
)

// InitProjectTool initializes a new Databricks project.
var InitProjectTool = Tool{
	Definition: ToolDefinition{
		Name:        "init_project",
		Description: "Initialize a new Databricks project structure. Use this to create a new project. After initialization, use add_project_resource to add specific resources (apps, jobs, pipelines, dashboards) as needed.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_name": map[string]any{
					"type":        "string",
					"description": "A name for this project in snake_case. Ask the user about this if it's not clear from the context.",
				},
				"project_path": map[string]any{
					"type":        "string",
					"description": "A fully qualified path of the project directory. Usually this should be in the current directory! But if it already has a lot of other things then it should be a subdirectory. Files will be created directly at this path.",
				},
				"language": map[string]any{
					"type":        "string",
					"description": "Language: 'python' (includes pyproject.toml) or 'other' (recommended for apps). Default: 'python'.",
					"enum":        []string{"python", "other"},
				},
			},
			"required": []string{"project_name", "project_path"},
		},
	},
	Handler: func(ctx context.Context, args map[string]any) (string, error) {
		var typedArgs initProjectArgs
		if err := UnmarshalArgs(args, &typedArgs); err != nil {
			return "", err
		}
		return InitProject(ctx, typedArgs)
	},
}

type initProjectArgs struct {
	ProjectName string `json:"project_name"`
	ProjectPath string `json:"project_path"`
	Language    string `json:"language"`
}

// InitProject initializes a new Databricks project using the default-minimal template.
func InitProject(ctx context.Context, args initProjectArgs) (string, error) {
	if args.ProjectPath == "" {
		return "", errors.New("project_path is required")
	}
	if args.ProjectName == "" {
		return "", errors.New("project_name is required")
	}
	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
	}

	if err := os.MkdirAll(args.ProjectPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}

	if _, err := os.Stat(filepath.Join(args.ProjectPath, "databricks.yml")); err == nil {
		return "", fmt.Errorf("project already initialized: databricks.yml exists in %s\n\nUse the add_project_resource tool to add resources to this existing project", args.ProjectPath)
	}

	if args.Language == "" {
		args.Language = "python"
	}

	if err := runBundleInit(ctx, args.ProjectPath, args.ProjectName, args.Language); err != nil {
		return "", err
	}

	// Template creates a nested directory - move contents to project root
	nestedPath := filepath.Join(args.ProjectPath, args.ProjectName)
	entries, err := os.ReadDir(nestedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read nested project directory: %w", err)
	}

	for _, entry := range entries {
		if err := os.Rename(filepath.Join(nestedPath, entry.Name()), filepath.Join(args.ProjectPath, entry.Name())); err != nil {
			return "", fmt.Errorf("failed to move %s: %w", entry.Name(), err)
		}
	}
	os.Remove(nestedPath)

	filename := "AGENTS.md"
	if strings.Contains(strings.ToLower(GetClientName(ctx)), "claude") {
		filename = "CLAUDE.md"
	}

	templateData := map[string]string{
		"ProjectName": args.ProjectName,
		"ProjectPath": args.ProjectPath,
	}
	instructionsContent := prompts.MustExecuteTemplate("AGENTS.tmpl", templateData)
	if err := os.WriteFile(filepath.Join(args.ProjectPath, filename), []byte(instructionsContent), 0o644); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", filename, err)
	}

	analysis, err := AnalyzeProject(ctx, analyzeProjectArgs{ProjectPath: args.ProjectPath})
	if err != nil {
		return "", fmt.Errorf("failed to analyze initialized project: %w", err)
	}

	return instructionsContent + analysis, nil
}

func runBundleInit(ctx context.Context, projectPath, projectName, language string) error {
	configJSON, err := json.Marshal(map[string]string{
		"project_name":     projectName,
		"default_catalog":  "main",
		"personal_schemas": "yes",
		"language_choice":  language,
	})
	if err != nil {
		return fmt.Errorf("failed to create config JSON: %w", err)
	}

	configFile := filepath.Join(os.TempDir(), fmt.Sprintf("databricks-init-%s.json", projectName))
	if err := os.WriteFile(configFile, configJSON, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	defer os.Remove(configFile)

	cmd := exec.CommandContext(ctx, GetCLIPath(), "bundle", "init",
		"--config-file", configFile,
		"--output-dir", projectPath,
		"default-minimal")

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to initialize project: %w\nOutput: %s", err, string(output))
	}

	return nil
}
