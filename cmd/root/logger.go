package root

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/bricks/libs/log"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

const (
	envBricksLogFile   = "BRICKS_LOG_FILE"
	envBricksLogLevel  = "BRICKS_LOG_LEVEL"
	envBricksLogFormat = "BRICKS_LOG_FORMAT"
)

func initializeLogger(ctx context.Context, cmd *cobra.Command) (context.Context, error) {
	opts := slog.HandlerOptions{}
	opts.Level = flags.LogLevel.Level()
	opts.AddSource = true
	opts.ReplaceAttr = log.ReplaceAttrFunctions{
		log.ReplaceLevelAttr,
		log.ReplaceSourceAttr,
	}.ReplaceAttr

	// Open the underlying log file if the user configured an actual file to log to.
	err := flags.LogFile.Open()
	if err != nil {
		return nil, err
	}

	var handler slog.Handler
	switch logOutput {
	case flags.OutputJSON:
		handler = opts.NewJSONHandler(flags.LogFile.Writer())
	case flags.OutputText:
		handler = opts.NewTextHandler(flags.LogFile.Writer())
	default:
		return nil, fmt.Errorf("invalid log output: %s", logOutput)
	}

	slog.SetDefault(slog.New(handler))
	return log.NewContext(ctx, slog.Default()), nil
}

var logOutput = flags.OutputText

func init() {
	// Configure defaults from environment, if applicable.
	// If the provided value is invalid it is ignored.
	if v, ok := os.LookupEnv(envBricksLogFile); ok {
		flags.LogFile.Set(v)
	}
	if v, ok := os.LookupEnv(envBricksLogLevel); ok {
		flags.LogLevel.Set(v)
	}
	if v, ok := os.LookupEnv(envBricksLogFormat); ok {
		logOutput.Set(v)
	}

	RootCmd.PersistentFlags().Var(&flags.LogFile, "log-file", "file to write logs to")
	RootCmd.PersistentFlags().Var(&flags.LogLevel, "log-level", "log level")
	RootCmd.PersistentFlags().Var(&logOutput, "log-format", "log output format (text or json)")
	RootCmd.RegisterFlagCompletionFunc("log-file", flags.LogFile.Complete)
	RootCmd.RegisterFlagCompletionFunc("log-level", flags.LogLevel.Complete)
	RootCmd.RegisterFlagCompletionFunc("log-format", logOutput.Complete)
}
