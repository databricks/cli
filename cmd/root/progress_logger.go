package root

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/bricks/libs/progress"
	"golang.org/x/term"
)

const envBricksProgressFormat = "BRICKS_PROGRESS_FORMAT"

func resolveModeDefault(format flags.ProgressLogFormat) flags.ProgressLogFormat {
	if (logLevel.String() == "disabled" || logFile.String() != "stderr") &&
		term.IsTerminal(int(os.Stderr.Fd())) {
		return flags.ModeInplace
	}
	return flags.ModeAppend
}

func initializeProgressLogger(ctx context.Context) (context.Context, error) {
	if logLevel.String() != "disabled" && logFile.String() == "stderr" &&
		progressFormat == flags.ModeInplace {
		return nil, fmt.Errorf("inplace progress logging cannot be used when log-file is stderr")
	}

	format := progressFormat
	if format == flags.ModeDefault {
		format = resolveModeDefault(format)
	}

	progressLogger := progress.NewLogger(format)
	return progress.NewContext(ctx, progressLogger), nil
}

var progressFormat = flags.NewProgressLogFormat()

// TODO: configure autocomplete
func init() {
	// Configure defaults from environment, if applicable.
	// If the provided value is invalid it is ignored.
	if v, ok := os.LookupEnv(envBricksProgressFormat); ok {
		progressFormat.Set(v)
	}
	RootCmd.PersistentFlags().Var(&progressFormat, "progress-format", "format for progress logs (append, inplace, json)")
	RootCmd.RegisterFlagCompletionFunc("progress-format", progressFormat.Complete)
}
