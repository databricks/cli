package root

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/exp/slog"
)

const (
	envDatabricksCliLogFile  = "DATABRICKS_CLI_LOG_FILE"
	envDatabricksCliLogLevel = "DATABRICKS_CLI_LOG_LEVEL"
	envDataricksCliLogFormat = "DATABRICKS_CLI_LOG_FORMAT"
)

func initializeLogger(ctx context.Context) (context.Context, error) {
	opts := slog.HandlerOptions{}
	opts.Level = logLevel.Level()
	opts.AddSource = true
	opts.ReplaceAttr = log.ReplaceAttrFunctions{
		log.ReplaceLevelAttr,
		log.ReplaceSourceAttr,
	}.ReplaceAttr

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
		return nil, fmt.Errorf("invalid log output: %s", logOutput)
	}

	slog.SetDefault(slog.New(handler))
	return log.NewContext(ctx, slog.Default()), nil
}

var logFile = flags.NewLogFileFlag()
var logLevel = flags.NewLogLevelFlag()
var logOutput = flags.OutputText

func init() {
	// Configure defaults from environment, if applicable.
	// If the provided value is invalid it is ignored.
	if v, ok := os.LookupEnv(envDatabricksCliLogFile); ok {
		logFile.Set(v)
	}
	if v, ok := os.LookupEnv(envDatabricksCliLogLevel); ok {
		logLevel.Set(v)
	}
	if v, ok := os.LookupEnv(envDataricksCliLogFormat); ok {
		logOutput.Set(v)
	}

	RootCmd.PersistentFlags().Var(&logFile, "log-file", "file to write logs to")
	RootCmd.PersistentFlags().Var(&logLevel, "log-level", "log level")
	RootCmd.PersistentFlags().Var(&logOutput, "log-format", "log output format (text or json)")
	RootCmd.RegisterFlagCompletionFunc("log-file", logFile.Complete)
	RootCmd.RegisterFlagCompletionFunc("log-level", logLevel.Complete)
	RootCmd.RegisterFlagCompletionFunc("log-format", logOutput.Complete)
}
