package init_template

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

// newPipelineCmd creates the pipeline subcommand for init-template.
func newPipelineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Initialize a Lakeflow pipeline project",
		Args:  cobra.NoArgs,
		Long: `Initialize a Lakeflow Declarative Pipeline project.

This creates a project with:
- Pipeline definitions in src/ directory (Python or SQL)
- Pipeline configuration in resources/ using databricks.yml
- Serverless compute enabled by default
- Personal schemas for development

Examples:
  experimental apps-mcp tools init-template pipeline --name my_pipeline --language python
  experimental apps-mcp tools init-template pipeline --name my_pipeline --language sql
  experimental apps-mcp tools init-template pipeline --name my_pipeline --language python --catalog my_catalog
  experimental apps-mcp tools init-template pipeline --name my_pipeline --language sql --output-dir ./projects

After initialization:
  databricks bundle deploy --target dev
`,
	}

	var name string
	var language string
	var catalog string
	var outputDir string

	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&language, "language", "", "Pipeline language: 'python' or 'sql' (required)")
	cmd.Flags().StringVar(&catalog, "catalog", "", "Default catalog for tables (defaults to workspace default)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if name == "" {
			return errors.New("--name is required. Example: init-template pipeline --name my_pipeline --language python")
		}
		if language == "" {
			return errors.New("--language is required. Choose 'python' or 'sql'. Example: init-template pipeline --name my_pipeline --language python")
		}
		if language != "python" && language != "sql" {
			return fmt.Errorf("--language must be 'python' or 'sql', got '%s'", language)
		}

		// Default to workspace default catalog if not specified
		if catalog == "" {
			catalog = middlewares.GetDefaultCatalog(ctx)
		}

		configMap := map[string]any{
			"project_name":     name,
			"personal_schemas": "yes",
			"language":         language,
			"default_catalog":  catalog,
		}

		return MaterializeTemplate(ctx, TemplateConfig{
			TemplatePath: string(template.LakeflowPipelines),
			TemplateName: "lakeflow-pipelines",
		}, configMap, name, outputDir)
	}
	return cmd
}
