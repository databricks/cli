// Copied from cmd/root/progress_logger.go and adapted for pipelines use.
package root

import (
	"context"
	"errors"
	"os"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const envProgressFormat = "DATABRICKS_CLI_PROGRESS_FORMAT"

type progressLoggerFlag struct {
	flags.ProgressLogFormat

	log *logFlags
}

func (f *progressLoggerFlag) resolveModeDefault(format flags.ProgressLogFormat) flags.ProgressLogFormat {
	if (f.log.level.String() == "disabled" || f.log.file.String() != "stderr") &&
		term.IsTerminal(int(os.Stderr.Fd())) {
		return flags.ModeInplace
	}
	return flags.ModeAppend
}

func (f *progressLoggerFlag) initializeContext(ctx context.Context) (context.Context, error) {
	// No need to initialize the logger if it's already set in the context. This
	// happens in unit tests where the logger is setup as a fixture.
	if _, ok := cmdio.FromContext(ctx); ok {
		return ctx, nil
	}

	if f.log.level.String() != "disabled" && f.log.file.String() == "stderr" &&
		f.ProgressLogFormat == flags.ModeInplace {
		return nil, errors.New("inplace progress logging cannot be used when log-file is stderr")
	}

	format := f.ProgressLogFormat
	if format == flags.ModeDefault {
		format = f.resolveModeDefault(format)
	}

	progressLogger := cmdio.NewLogger(format)
	return cmdio.NewContext(ctx, progressLogger), nil
}

func initProgressLoggerFlag(cmd *cobra.Command, logFlags *logFlags) *progressLoggerFlag {
	f := progressLoggerFlag{
		ProgressLogFormat: flags.NewProgressLogFormat(),

		log: logFlags,
	}

	// Configure defaults from environment, if applicable.
	// If the provided value is invalid it is ignored.
	if v, ok := env.Lookup(cmd.Context(), envProgressFormat); ok {
		_ = f.Set(v)
	}

	flags := cmd.PersistentFlags()
	flags.Var(&f.ProgressLogFormat, "progress-format", "format for progress logs (append, inplace, json)")
	flags.MarkHidden("progress-format")
	cmd.RegisterFlagCompletionFunc("progress-format", f.ProgressLogFormat.Complete)
	return &f
}
