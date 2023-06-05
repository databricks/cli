package root

import (
	"os"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

const envOutputFormat = "DATABRICKS_OUTPUT_FORMAT"

var outputType flags.Output = flags.OutputText

func init() {
	// Configure defaults from environment, if applicable.
	// If the provided value is invalid it is ignored.
	if v, ok := os.LookupEnv(envOutputFormat); ok {
		outputType.Set(v)
	}

	RootCmd.PersistentFlags().VarP(&outputType, "output", "o", "output type: text or json")
}

func OutputType() flags.Output {
	return outputType
}

func initializeIO(cmd *cobra.Command) error {
	var template string
	if cmd.Annotations != nil {
		// rely on zeroval being an empty string
		template = cmd.Annotations["template"]
	}

	cmdIO := cmdio.NewIO(outputType, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), template)
	ctx := cmdio.InContext(cmd.Context(), cmdIO)
	cmd.SetContext(ctx)

	return nil
}
