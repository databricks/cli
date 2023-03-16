package root

import (
	"context"

	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/bricks/libs/logger"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

func initializeLogger(ctx context.Context, cmd *cobra.Command) (context.Context, error) {
	opts := slog.HandlerOptions{}
	opts.Level = logLevel.Level()
	opts.AddSource = true
	opts.ReplaceAttr = logger.ReplaceLevelAttr

	// Open the underlying log file if the user configured an actual file to log to.
	err := logFile.Open()
	if err != nil {
		return nil, err
	}

	var handler slog.Handler
	switch logOutput {
	case flags.OutputJSON:
		handler = opts.NewJSONHandler(logFile.Writer())
	case flags.OutputText:
		handler = opts.NewTextHandler(logFile.Writer())
	default:
		panic("invalid log output: " + logOutput)
	}

	slog.SetDefault(slog.New(handler))
	return logger.NewContext(ctx, slog.Default()), nil
}

var logFile = flags.NewLogFileFlag()
var logLevel = flags.NewLogLevelFlag()
var logOutput = flags.OutputText

func init() {
	RootCmd.PersistentFlags().Var(&logFile, "log-file", "file to write logs to")
	RootCmd.PersistentFlags().Var(&logLevel, "log-level", "log level")
	RootCmd.PersistentFlags().Var(&logOutput, "log-format", "log output format (text or json)")
	RootCmd.RegisterFlagCompletionFunc("log-file", logFile.Complete)
	RootCmd.RegisterFlagCompletionFunc("log-level", logLevel.Complete)
	RootCmd.RegisterFlagCompletionFunc("log-format", logOutput.Complete)
}
