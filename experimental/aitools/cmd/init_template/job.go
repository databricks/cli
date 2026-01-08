package init_template

import (
	"errors"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/aitools/lib/middlewares"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

// newJobCmd creates the job subcommand for init-template.
func newJobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Initialize a job project using the default-python template",
		Args:  cobra.NoArgs,
		Long: `Initialize a job project using the default-python template.

This creates a project with:
- Python notebooks in src/ directory
- A wheel package defined in pyproject.toml
- Job definitions in resources/ using databricks.yml
- Serverless compute enabled by default
- Personal schemas for development

Examples:
  experimental apps-mcp tools init-template job --name my_job
  experimental apps-mcp tools init-template job --name my_job --catalog my_catalog
  experimental apps-mcp tools init-template job --name my_job --output-dir ./projects

After initialization:
  databricks bundle deploy --target dev
`,
	}

	var name string
	var catalog string
	var outputDir string

	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&catalog, "catalog", "", "Default catalog for tables (defaults to workspace default)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if name == "" {
			return errors.New("--name is required. Example: init-template job --name my_job")
		}

		// Default to workspace default catalog if not specified
		if catalog == "" {
			catalog = middlewares.GetDefaultCatalog(ctx)
		}

		configMap := map[string]any{
			"project_name":     name,
			"include_job":      "yes",
			"include_pipeline": "no",
			"include_python":   "yes",
			"serverless":       "yes",
			"personal_schemas": "yes",
			"default_catalog":  catalog,
		}

		return MaterializeTemplate(ctx, TemplateConfig{
			TemplatePath: string(template.DefaultPython),
			TemplateName: "default-python",
		}, configMap, name, outputDir)
	}
	return cmd
}
