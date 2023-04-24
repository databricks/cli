package root

import (
	"os"

	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/bricks/libs/flags"
	"github.com/spf13/cobra"
)

const envBricksOutputFormat = "BRICKS_OUTPUT_FORMAT"

var outputType flags.Output = flags.OutputText

func init() {
	RootCmd.PersistentFlags().VarP(&outputType, "output", "o", "output type: text or json")
}

func OutputType() flags.Output {
	return outputType
}

func initializeIO(cmd *cobra.Command) error {
	output := os.Getenv(envBricksOutputFormat)
	if output != "" {
		err := outputType.Set(output)
		if err != nil {
			return err
		}
	}

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
