package root

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

const envOutputFormat = "DATABRICKS_OUTPUT_FORMAT"
const envQuiet = "DATABRICKS_QUIET"
const envNoInput = "DATABRICKS_NO_INPUT"
const envYes = "DATABRICKS_YES"

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
	if v, ok := env.Lookup(ctx, envQuiet); ok && isTruthy(v) {
		f.quiet = true
	}
	if v, ok := env.Lookup(ctx, envNoInput); ok && isTruthy(v) {
		f.noInput = true
	}
	if v, ok := env.Lookup(ctx, envYes); ok && isTruthy(v) {
		f.yes = true
	}

	cmd.PersistentFlags().BoolVarP(&f.quiet, "quiet", "q", f.quiet, "Suppress non-essential output. Use with --yes for fully silent operation.")
	cmd.PersistentFlags().BoolVar(&f.noInput, "no-input", f.noInput, "disable interactive prompts")
	cmd.PersistentFlags().BoolVarP(&f.yes, "yes", "y", f.yes, "auto-approve all yes/no prompts")
	return f
}

// isTruthy returns true for common truthy string values.
func isTruthy(v string) bool {
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
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

func (f *interactionFlags) applyToContext(ctx context.Context) {
	if f.quiet {
		cmdio.SetQuiet(ctx)
	}
	if f.noInput {
		cmdio.SetNoInput(ctx)
	}
	if f.yes {
		cmdio.SetYes(ctx)
	}
}
