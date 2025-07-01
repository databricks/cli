package root

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
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
	if v, ok := env.Lookup(cmd.Context(), envOutputFormat); ok {
		f.output.Set(v) //nolint:errcheck
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
	var headerTemplate, template string
	if cmd.Annotations != nil {
		// rely on zeroval being an empty string
		template = cmd.Annotations["template"]
		headerTemplate = cmd.Annotations["headerTemplate"]
	}

	ctx := cmd.Context()
	cmdIO := cmdio.NewIO(ctx, f.output, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), headerTemplate, template)
	ctx = cmdio.InContext(ctx, cmdIO)
	cmd.SetContext(ctx)
	return nil
}
