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

		configMap := map[string]any{
			"project_name":     name,
			"lakeflow_only":    "yes",
			"include_job":      "no",
			"include_pipeline": "yes",
			"include_python":   "no",
			"serverless":       "yes",
			"personal_schemas": "yes",
			"language":         language,
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
			TemplatePathOrUrl: string(template.LakeflowPipelines),
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
		extraDetails := "Language: " + language
		cmdio.LogString(ctx, common.FormatProjectScaffoldSuccess("pipeline", "🔄", "lakeflow-pipelines", absOutputDir, fileCount, extraDetails))

		fileTree, err := generateFileTree(absOutputDir)
		if err == nil && fileTree != "" {
			cmdio.LogString(ctx, "\nFile structure:")
			cmdio.LogString(ctx, fileTree)
		}

		// Write CLAUDE.md and AGENTS.md files
		if err := writeAgentFiles(absOutputDir, map[string]any{}); err != nil {
			return fmt.Errorf("failed to write agent files: %w", err)
		}

		// Show L2 guidance: mixed (for adding any resource) + pipelines (for developing pipelines)
		targetMixed := prompts.MustExecuteTemplate("target_mixed.tmpl", map[string]any{})
		cmdio.LogString(ctx, targetMixed)

		targetPipelines := prompts.MustExecuteTemplate("target_pipelines.tmpl", map[string]any{})
		cmdio.LogString(ctx, targetPipelines)

		return nil
	}
	return cmd
}
