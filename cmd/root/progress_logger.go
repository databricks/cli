package root

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/bricks/libs/flags"
	"golang.org/x/term"
)

const envBricksProgressFormat = "BRICKS_PROGRESS_FORMAT"

// Inplace logging is supported as default only if debug logger does not log to
// stderr and stderr is a tty
func isInplaceSupported() bool {
	return (logLevel.String() == "disabled" || logFile.String() != "stderr") &&
		term.IsTerminal(int(os.Stderr.Fd()))
}

func initializeProgressLogger(ctx context.Context) (context.Context, error) {
	if logLevel.String() != "disabled" && logFile.String() == "stderr" &&
		progressFormat == flags.ModeInplace {
		return nil, fmt.Errorf("inplace progress logging cannot be used when log-file is stderr")
	}

	progressLogger := cmdio.NewLogger(progressFormat, isInplaceSupported())
	return cmdio.NewContext(ctx, progressLogger), nil
}

var progressFormat = flags.NewProgressLogFormat()

func init() {
	// Configure defaults from environment, if applicable.
	// If the provided value is invalid it is ignored.
	if v, ok := os.LookupEnv(envBricksProgressFormat); ok {
		progressFormat.Set(v)
	}
	RootCmd.PersistentFlags().Var(&progressFormat, "progress-format", "format for progress logs (append, inplace, json)")
	RootCmd.RegisterFlagCompletionFunc("progress-format", progressFormat.Complete)
}
