package root

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/bricks/libs/progress"
	"golang.org/x/term"
)

// TODO: setup env vars

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

// TODO: setup and test autocomplete
func init() {
	RootCmd.PersistentFlags().Var(&progressFormat, "progress-format", "format for progress logs (append, inplace, json)")
}
