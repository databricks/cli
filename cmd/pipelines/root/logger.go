// Copied from cmd/root/logger.go and adapted for pipelines use.
package root

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/log/handler"
	"github.com/spf13/cobra"
)

const (
	envLogFile   = "DATABRICKS_LOG_FILE"
	envLogLevel  = "DATABRICKS_LOG_LEVEL"
	envLogFormat = "DATABRICKS_LOG_FORMAT"
)

type logFlags struct {
	file   flags.LogFileFlag
	level  flags.LogLevelFlag
	output flags.Output
	debug  bool
}

func (f *logFlags) makeLogHandler(opts slog.HandlerOptions) (slog.Handler, error) {
	switch f.output {
	case flags.OutputJSON:
		return slog.NewJSONHandler(f.file.Writer(), &opts), nil
	case flags.OutputText:
		w := f.file.Writer()
		return handler.NewFriendlyHandler(w, &handler.Options{
			Color:       cmdio.IsTTY(w),
			Level:       opts.Level,
			ReplaceAttr: opts.ReplaceAttr,
		}), nil
	default:
		return nil, fmt.Errorf("invalid log output mode: %s", f.output)
	}
}

func (f *logFlags) initializeContext(ctx context.Context) (context.Context, error) {
	if f.debug {
		err := f.level.Set("debug")
		if err != nil {
			return nil, err
		}
	}

	opts := slog.HandlerOptions{}
	opts.Level = f.level.Level()
	opts.AddSource = true
	opts.ReplaceAttr = log.ReplaceAttrFunctions{
		log.ReplaceLevelAttr,
		log.ReplaceSourceAttr,
	}.ReplaceAttr

	// Open the underlying log file if the user configured an actual file to log to.
	err := f.file.Open()
	if err != nil {
		return nil, err
	}

	handler, err := f.makeLogHandler(opts)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(slog.New(handler).With(slog.Int("pid", os.Getpid())))
	return log.NewContext(ctx, slog.Default()), nil
}

func initLogFlags(cmd *cobra.Command) *logFlags {
	f := logFlags{
		file:   flags.NewLogFileFlag(),
		level:  flags.NewLogLevelFlag(),
		output: flags.OutputText,
	}

	// Configure defaults from environment, if applicable.
	// If the provided value is invalid it is ignored.
	if v, ok := env.Lookup(cmd.Context(), envLogFile); ok {
		f.file.Set(v) //nolint:errcheck
	}
	if v, ok := env.Lookup(cmd.Context(), envLogLevel); ok {
		f.level.Set(v) //nolint:errcheck
	}
	if v, ok := env.Lookup(cmd.Context(), envLogFormat); ok {
		f.output.Set(v) //nolint:errcheck
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&f.debug, "debug", false, "enable debug logging")
	flags.Var(&f.file, "log-file", "file to write logs to")
	flags.Var(&f.level, "log-level", "log level")
	flags.Var(&f.output, "log-format", "log output format (text or json)")

	// mark fine-grained flags hidden from global --help
	flags.MarkHidden("log-file")
	flags.MarkHidden("log-level")
	flags.MarkHidden("log-format")

	cmd.RegisterFlagCompletionFunc("log-file", f.file.Complete)
	cmd.RegisterFlagCompletionFunc("log-level", f.level.Complete)
	cmd.RegisterFlagCompletionFunc("log-format", f.output.Complete)
	return &f
}
