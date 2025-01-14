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
		Long:  "Generate bundle configuration",
	}

	cmd.AddCommand(generate.NewGenerateJobCommand())
	cmd.AddCommand(generate.NewGeneratePipelineCommand())
	cmd.AddCommand(generate.NewGenerateDashboardCommand())
	cmd.AddCommand(generate.NewGenerateAppCommand())
	cmd.PersistentFlags().StringVar(&key, "key", "", `resource key to use for the generated configuration`)
	return cmd
}
