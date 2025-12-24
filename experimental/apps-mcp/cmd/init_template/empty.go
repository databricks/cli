package init_template

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/apps-mcp/lib/common"
	"github.com/databricks/cli/experimental/apps-mcp/lib/prompts"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

// newEmptyCmd creates the empty subcommand for init-template.
func newEmptyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "empty",
		Short: "Initialize an empty project for custom resources",
		Args:  cobra.NoArgs,
		Long: `Initialize an empty Databricks Asset Bundle project.

Use this for deploying resource types OTHER than apps, jobs, or pipelines, such as:
- Dashboards (Lakeview dashboards)
- Alerts (SQL alerts)
- Model serving endpoints
- Clusters
- Schemas and tables
- Any other Databricks resources

This creates a minimal project structure without sample code. For apps, jobs, or pipelines,
use the dedicated 'app', 'job', or 'pipeline' commands instead.

Examples:
  experimental apps-mcp tools init-template empty --name my_dashboard_project
  experimental apps-mcp tools init-template empty --name my_alerts --language sql --catalog my_catalog
  experimental apps-mcp tools init-template empty --name my_project --output-dir ./projects

After initialization:
  Add resource definitions in resources/ (e.g., resources/my_dashboard.dashboard.yml)
  Then deploy: databricks bundle deploy --target dev
`,
	}

	var name string
	var catalog string
	var language string
	var outputDir string

	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&catalog, "catalog", "", "Default catalog for tables (defaults to workspace default)")
	cmd.Flags().StringVar(&language, "language", "python", "Initial language: 'python', 'sql', or 'other'")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if name == "" {
			return errors.New("--name is required. Example: init-template empty --name my_project")
		}

		if language != "python" && language != "sql" && language != "other" {
			return fmt.Errorf("--language must be 'python', 'sql', or 'other', got '%s'", language)
		}

		configMap := map[string]any{
			"project_name":     name,
			"include_job":      "no",
			"include_pipeline": "no",
			"include_python":   "no",
			"serverless":       "yes",
			"personal_schemas": "yes",
			"language_choice":  language,
			"lakeflow_only":    "no",
			"enable_pydabs":    "no",
		}
		if catalog != "" {
			configMap["default_catalog"] = catalog
		}

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
			TemplatePathOrUrl: string(template.DefaultMinimal),
			ConfigFile:        configFile,
			OutputDir:         outputDir,
		}

		tmpl, err := r.Resolve(ctx)
		if err != nil {
			return err
		}
		defer tmpl.Reader.Cleanup(ctx)

		err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
		if err != nil {
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
		cmdio.LogString(ctx, common.FormatProjectScaffoldSuccess("empty", "📦", "default-minimal", absOutputDir, fileCount, ""))

		fileTree, err := generateFileTree(absOutputDir)
		if err == nil && fileTree != "" {
			cmdio.LogString(ctx, "\nFile structure:")
			cmdio.LogString(ctx, fileTree)
		}

		// Write CLAUDE.md and AGENTS.md files
		if err := writeAgentFiles(absOutputDir, map[string]any{}); err != nil {
			return fmt.Errorf("failed to write agent files: %w", err)
		}

		targetMixed := prompts.MustExecuteTemplate("target_mixed.tmpl", map[string]any{})
		cmdio.LogString(ctx, targetMixed)

		return nil
	}
	return cmd
}
