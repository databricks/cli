package bundle

import (
	"github.com/databricks/cli/cmd/bundle/generate"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/spf13/cobra"
)

func newGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate bundle configuration",
		Long:    "Generate bundle configuration",
		PreRunE: utils.ConfigureBundleWithVariables,
	}

	cmd.AddCommand(generate.NewGenerateJobCommand())
	return cmd
}
