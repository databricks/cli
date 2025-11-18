package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/aitools/auth"
	"github.com/databricks/cli/experimental/aitools/tools/prompts"
	"github.com/databricks/cli/experimental/aitools/tools/resources"
)

// AnalyzeProjectTool analyzes a Databricks project and returns guidance.
// It uses hardcoded guidance + guidance from the project's README.md file for this.
var AnalyzeProjectTool = Tool{
	Definition: ToolDefinition{
		Name:        "analyze_project",
		Description: "MANDATORY - REQUIRED FIRST STEP: If databricks.yml exists in the directory, you MUST call this tool before using Read, Glob, or any other tools. Databricks projects require specialized commands that differ from standard Python/Node.js workflows - attempting standard approaches will fail. This tool is fast and provides the correct commands for preview/deploy/run operations.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_path": map[string]any{
					"type":        "string",
					"description": "A fully qualified path of the project to operate on. By default, the current directory.",
				},
			},
			"required": []string{"project_path"},
		},
	},
	Handler: func(ctx context.Context, args map[string]any) (string, error) {
		var typedArgs analyzeProjectArgs
		if err := UnmarshalArgs(args, &typedArgs); err != nil {
			return "", err
		}
		return AnalyzeProject(ctx, typedArgs)
	},
}

type analyzeProjectArgs struct {
	ProjectPath string `json:"project_path"`
}

// AnalyzeProject analyzes a Databricks project and returns information about it.
func AnalyzeProject(ctx context.Context, args analyzeProjectArgs) (string, error) {
	if err := ValidateDatabricksProject(args.ProjectPath); err != nil {
		return "", err
	}

	if err := auth.CheckAuthentication(ctx); err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, GetCLIPath(), "bundle", "summary")
	cmd.Dir = args.ProjectPath

	output, err := cmd.CombinedOutput()
	var summary string
	if err != nil {
		summary = "Bundle summary failed:\n" + string(output)
	} else {
		summary = string(output)
	}

	var readmeContent string
	if content, err := os.ReadFile(filepath.Join(args.ProjectPath, "README.md")); err == nil {
		readmeContent = "\n\nProject-Specific Guidance\n" +
			"-------------------------\n" +
			string(content)
	}

	// Get default warehouse for apps and other resources that need it
	warehouse, err := GetDefaultWarehouse(ctx)
	resourceGuidance := getResourceGuidance(args.ProjectPath, warehouse)

	data := map[string]string{
		"Summary":          summary,
		"ReadmeContent":    readmeContent,
		"ResourceGuidance": resourceGuidance,
	}

	if err == nil && warehouse != nil {
		data["WarehouseID"] = warehouse.ID
		data["WarehouseName"] = warehouse.Name
	}

	result := prompts.MustExecuteTemplate("analyze_project.tmpl", data)

	return result, nil
}

// getResourceGuidance scans the resources directory and collects guidance for detected resource types.
func getResourceGuidance(projectPath string, warehouse *warehouse) string {
	var guidance strings.Builder

	// Extract warehouse ID and name for resource handlers
	warehouseID := ""
	warehouseName := ""
	if warehouse != nil {
		warehouseID = warehouse.ID
		warehouseName = warehouse.Name
	}

	detected := make(map[string]bool)

	// Check resources directory
	resourcesDir := filepath.Join(projectPath, "resources")
	if entries, err := os.ReadDir(resourcesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			var resourceType string
			switch {
			case strings.HasSuffix(name, ".app.yml") || strings.HasSuffix(name, ".app.yaml"):
				resourceType = "app"
			case strings.HasSuffix(name, ".job.yml") || strings.HasSuffix(name, ".job.yaml"):
				resourceType = "job"
			case strings.HasSuffix(name, ".pipeline.yml") || strings.HasSuffix(name, ".pipeline.yaml"):
				resourceType = "pipeline"
			case strings.HasSuffix(name, ".dashboard.yml") || strings.HasSuffix(name, ".dashboard.yaml"):
				resourceType = "dashboard"
			default:
				continue
			}
			if !detected[resourceType] {
				detected[resourceType] = true
				handler := resources.GetResourceHandler(resourceType)
				if handler != nil {
					guidanceText := handler.GetGuidancePrompt(projectPath, warehouseID, warehouseName)
					if guidanceText != "" {
						guidance.WriteString(guidanceText)
						guidance.WriteString("\n")
					}
				}
			}
		}
	}
	return guidance.String()
}
