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

		configMap := map[string]any{
			"project_name":     name,
			"include_job":      "yes",
			"include_pipeline": "no",
			"include_python":   "yes",
			"serverless":       "yes",
			"personal_schemas": "yes",
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
			TemplatePathOrUrl: string(template.DefaultPython),
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
		cmdio.LogString(ctx, common.FormatProjectScaffoldSuccess("job", "⚙️", "default-python", absOutputDir, fileCount, ""))

		fileTree, err := generateFileTree(absOutputDir)
		if err == nil && fileTree != "" {
			cmdio.LogString(ctx, "\nFile structure:")
			cmdio.LogString(ctx, fileTree)
		}

		// Write CLAUDE.md and AGENTS.md files
		if err := writeAgentFiles(absOutputDir, map[string]any{}); err != nil {
			return fmt.Errorf("failed to write agent files: %w", err)
		}

		// Show L2 guidance: mixed (for adding any resource) + jobs (for developing jobs)
		targetMixed := prompts.MustExecuteTemplate("target_mixed.tmpl", map[string]any{})
		cmdio.LogString(ctx, targetMixed)

		targetJobs := prompts.MustExecuteTemplate("target_jobs.tmpl", map[string]any{})
		cmdio.LogString(ctx, targetJobs)

		return nil
	}
	return cmd
}
