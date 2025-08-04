package bundle

import (
	"github.com/databricks/cli/cmd/bundle/generate"
	"github.com/spf13/cobra"
)

func newGenerateCommand() *cobra.Command {
	var key string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate bundle configuration",
		Long: `Generate bundle configuration from existing Databricks resources.

Common patterns:
  databricks bundle generate job --existing-job-id 123 --key my_job
  databricks bundle generate dashboard --existing-path /my-dashboard --key sales_dash
  databricks bundle generate dashboard --resource my_dashboard --watch  # Keep in sync

Complete migration workflow:
  1. Generate: databricks bundle generate job --existing-job-id 123 --key my_job
  2. Bind: databricks bundle deployment bind my_job 123
  3. Deploy: databricks bundle deploy

Use --key to specify the resource name in your bundle configuration.`,
	}

	cmd.AddCommand(generate.NewGenerateJobCommand())
	cmd.AddCommand(generate.NewGeneratePipelineCommand())
	cmd.AddCommand(generate.NewGenerateDashboardCommand())
	cmd.AddCommand(generate.NewGenerateAppCommand())
	cmd.PersistentFlags().StringVar(&key, "key", "", `resource key to use for the generated configuration`)
	return cmd
}
