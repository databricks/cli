package bundle

import (
	"github.com/databricks/cli/cmd/bundle/generate"
	"github.com/spf13/cobra"
)

func newGenerateCommand() *cobra.Command {
	var key string

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate bundle configuration",
		Long:    "Generate bundle configuration",
		PreRunE: ConfigureBundleWithVariables,
	}

	cmd.AddCommand(generate.NewGenerateJobCommand())
	cmd.AddCommand(generate.NewGeneratePipelineCommand())
	cmd.Flags().StringVar(&key, "key", "", `Key name to be used for generated resources in the config`)
	return cmd
}
