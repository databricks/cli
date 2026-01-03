package init_template

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

// newEmptyCmd creates the empty subcommand for init-template.
func newEmptyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "empty",
		Short: "Initialize an empty project for custom resources",
		Args:  cobra.NoArgs,
		Long: `Initialize an empty Databricks Asset Bundles project.

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

		// Default to workspace default catalog if not specified
		if catalog == "" {
			catalog = middlewares.GetDefaultCatalog(ctx)
		}

		configMap := map[string]any{
			"project_name":     name,
			"personal_schemas": "yes",
			"language_choice":  language,
			"default_catalog":  catalog,
		}

		return MaterializeTemplate(ctx, TemplateConfig{
			TemplatePath: string(template.DefaultMinimal),
			TemplateName: "default-minimal",
		}, configMap, name, outputDir)
	}
	return cmd
}
