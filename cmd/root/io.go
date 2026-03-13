package root

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

const (
	envOutputFormat = "DATABRICKS_OUTPUT_FORMAT"
	envQuiet        = "DATABRICKS_QUIET"
	envNoInput      = "DATABRICKS_NO_INPUT"
	envYes          = "DATABRICKS_YES"
)

type outputFlag struct {
	output flags.Output
}

type interactionFlags struct {
	quiet   bool
	noInput bool
	yes     bool
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

func initInteractionFlags(cmd *cobra.Command) *interactionFlags {
	f := &interactionFlags{}

	ctx := cmd.Context()

	// Configure defaults from environment variables.
	if enabled, ok := env.GetBool(ctx, envQuiet); ok && enabled {
		f.quiet = true
	}
	if enabled, ok := env.GetBool(ctx, envNoInput); ok && enabled {
		f.noInput = true
	}
	if enabled, ok := env.GetBool(ctx, envYes); ok && enabled {
		f.yes = true
	}

	cmd.PersistentFlags().BoolVarP(&f.quiet, "quiet", "q", f.quiet, "Suppress non-essential output")
	cmd.PersistentFlags().BoolVar(&f.noInput, "no-input", f.noInput, "Disable interactive prompts")
	cmd.PersistentFlags().BoolVarP(&f.yes, "yes", "y", f.yes, "Auto-approve all yes/no prompts")
	return f
}

func OutputType(cmd *cobra.Command) flags.Output {
	f, ok := cmd.Flag("output").Value.(*flags.Output)
	if !ok {
		panic("output flag not defined")
	}

	return *f
}

func (f *outputFlag) initializeIO(ctx context.Context, cmd *cobra.Command) (context.Context, error) {
	var headerTemplate, template string
	if cmd.Annotations != nil {
		// rely on zeroval being an empty string
		template = cmd.Annotations["template"]
		headerTemplate = cmd.Annotations["headerTemplate"]
	}

	cmdIO := cmdio.NewIO(ctx, f.output, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), headerTemplate, template)
	ctx = cmdio.InContext(ctx, cmdIO)
	cmd.SetContext(ctx)
	return ctx, nil
}

func (f *interactionFlags) applyToContext(ctx context.Context) context.Context {
	if f.quiet {
		ctx = cmdio.SetQuiet(ctx)
	}
	if f.noInput {
		ctx = cmdio.SetNoInput(ctx)
	}
	if f.yes {
		ctx = cmdio.SetYes(ctx)
	}
	return ctx
}
