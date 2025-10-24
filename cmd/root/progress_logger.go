package root

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

const envProgressFormat = "DATABRICKS_CLI_PROGRESS_FORMAT"

type progressLoggerFlag struct {
	flags.ProgressLogFormat
}

func (f *progressLoggerFlag) initializeContext(ctx context.Context) (context.Context, error) {
	// No need to initialize the logger if it's already set in the context. This
	// happens in unit tests where the logger is setup as a fixture.
	if _, ok := cmdio.FromContext(ctx); ok {
		return ctx, nil
	}

	format := f.ProgressLogFormat
	if format == flags.ModeDefault {
		format = flags.ModeAppend
	}

	progressLogger := cmdio.NewLogger(format)
	return cmdio.NewContext(ctx, progressLogger), nil
}

func initProgressLoggerFlag(cmd *cobra.Command, logFlags *logFlags) *progressLoggerFlag {
	f := progressLoggerFlag{
		ProgressLogFormat: flags.NewProgressLogFormat(),
	}

	// Configure defaults from environment, if applicable.
	// If the provided value is invalid it is ignored.
	if v, ok := env.Lookup(cmd.Context(), envProgressFormat); ok {
		_ = f.Set(v)
	}

	flags := cmd.PersistentFlags()
	flags.Var(&f.ProgressLogFormat, "progress-format", "format for progress logs (append)")
	flags.MarkHidden("progress-format")
	cmd.RegisterFlagCompletionFunc("progress-format", f.Complete)
	return &f
}
