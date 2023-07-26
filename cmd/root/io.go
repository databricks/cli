package root

import (
	"os"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

const envOutputFormat = "DATABRICKS_OUTPUT_FORMAT"

type outputFlag struct {
	output flags.Output
}

func initOutputFlag(cmd *cobra.Command) *outputFlag {
	f := outputFlag{
		output: flags.OutputText,
	}

	// Configure defaults from environment, if applicable.
	// If the provided value is invalid it is ignored.
	if v, ok := os.LookupEnv(envOutputFormat); ok {
		f.output.Set(v)
	}

	cmd.PersistentFlags().VarP(&f.output, "output", "o", "output type: text or json")
	return &f
}

func OutputType(cmd *cobra.Command) flags.Output {
	f, ok := cmd.Flag("output").Value.(*flags.Output)
	if !ok {
		panic("output flag not defined")
	}

	return *f
}

func (f *outputFlag) initializeIO(cmd *cobra.Command) error {
	var template string
	if cmd.Annotations != nil {
		// rely on zeroval being an empty string
		template = cmd.Annotations["template"]
	}

	cmdIO := cmdio.NewIO(f.output, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), template)
	ctx := cmdio.InContext(cmd.Context(), cmdIO)
	cmd.SetContext(ctx)
	return nil
}
