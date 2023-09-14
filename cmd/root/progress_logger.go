package root

import (
	"context"
	"fmt"
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
	if f.log.level.String() != "disabled" && f.log.file.String() == "stderr" &&
		f.ProgressLogFormat == flags.ModeInplace {
		return nil, fmt.Errorf("inplace progress logging cannot be used when log-file is stderr")
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
		f.Set(v)
	}

	cmd.PersistentFlags().Var(&f.ProgressLogFormat, "progress-format", "format for progress logs (append, inplace, json)")
	cmd.RegisterFlagCompletionFunc("progress-format", f.ProgressLogFormat.Complete)
	return &f
}
