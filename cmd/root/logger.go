package root

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	envLogFile   = "DATABRICKS_LOG_FILE"
	envLogLevel  = "DATABRICKS_LOG_LEVEL"
	envLogFormat = "DATABRICKS_LOG_FORMAT"
)

type friendlyHandler struct {
	slog.Handler
	w io.Writer
}

var (
	levelTrace = color.New(color.FgYellow).Sprint("TRACE")
	levelDebug = color.New(color.FgYellow).Sprint("DEBUG")
	levelInfo  = color.New(color.FgGreen).Sprintf("%5s", "INFO")
	levelWarn  = color.New(color.FgMagenta).Sprintf("%5s", "WARN")
	levelError = color.New(color.FgRed).Sprint("ERROR")
)

func (l *friendlyHandler) coloredLevel(rec slog.Record) string {
	switch rec.Level {
	case log.LevelTrace:
		return levelTrace
	case slog.LevelDebug:
		return levelDebug
	case slog.LevelInfo:
		return levelInfo
	case slog.LevelWarn:
		return levelWarn
	case log.LevelError:
		return levelError
	}
	return ""
}

func (l *friendlyHandler) Handle(ctx context.Context, rec slog.Record) error {
	t := fmt.Sprintf("%02d:%02d", rec.Time.Hour(), rec.Time.Minute())
	attrs := ""
	rec.Attrs(func(a slog.Attr) bool {
		attrs += fmt.Sprintf(" %s%s%s",
			color.CyanString(a.Key),
			color.CyanString("="),
			color.YellowString(a.Value.String()))
		return true
	})
	msg := fmt.Sprintf("%s %s %s%s\n",
		color.MagentaString(t),
		l.coloredLevel(rec),
		rec.Message,
		attrs)
	_, err := l.w.Write([]byte(msg))
	return err
}

type logFlags struct {
	file   flags.LogFileFlag
	level  flags.LogLevelFlag
	output flags.Output
}

func (f *logFlags) makeLogHandler(opts slog.HandlerOptions) (slog.Handler, error) {
	switch f.output {
	case flags.OutputJSON:
		return slog.NewJSONHandler(f.file.Writer(), &opts), nil
	case flags.OutputText:
		w := f.file.Writer()
		if cmdio.IsTTY(w) {
			return &friendlyHandler{
				Handler: slog.NewTextHandler(w, &opts),
				w:       w,
			}, nil
		}
		return slog.NewTextHandler(w, &opts), nil

	default:
		return nil, fmt.Errorf("invalid log output mode: %s", f.output)
	}
}

func (f *logFlags) initializeContext(ctx context.Context) (context.Context, error) {
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

	slog.SetDefault(slog.New(handler))
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
		f.file.Set(v)
	}
	if v, ok := env.Lookup(cmd.Context(), envLogLevel); ok {
		f.level.Set(v)
	}
	if v, ok := env.Lookup(cmd.Context(), envLogFormat); ok {
		f.output.Set(v)
	}

	cmd.PersistentFlags().Var(&f.file, "log-file", "file to write logs to")
	cmd.PersistentFlags().Var(&f.level, "log-level", "log level")
	cmd.PersistentFlags().Var(&f.output, "log-format", "log output format (text or json)")
	cmd.RegisterFlagCompletionFunc("log-file", f.file.Complete)
	cmd.RegisterFlagCompletionFunc("log-level", f.level.Complete)
	cmd.RegisterFlagCompletionFunc("log-format", f.output.Complete)
	return &f
}
