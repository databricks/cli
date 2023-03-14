package flags

import (
	"fmt"
	"strings"

	"github.com/databricks/bricks/libs/logger"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slog"
)

var levels = map[string]slog.Level{
	"trace":    logger.LevelTrace,
	"debug":    logger.LevelDebug,
	"info":     logger.LevelInfo,
	"warn":     logger.LevelWarn,
	"error":    logger.LevelError,
	"disabled": logger.LevelDisabled,
}

type LogLevelFlag struct {
	l slog.Level
}

func NewLogLevelFlag() LogLevelFlag {
	return LogLevelFlag{
		l: logger.LevelDisabled,
	}
}

func (f *LogLevelFlag) Level() slog.Level {
	return f.l
}

func (f *LogLevelFlag) String() string {
	for name, l := range levels {
		if f.l == l {
			return name
		}
	}

	return "(unknown)"
}

func (f *LogLevelFlag) Set(s string) error {
	l, ok := levels[strings.ToLower(s)]
	if !ok {
		return fmt.Errorf("accepted arguments are %s", strings.Join(maps.Keys(levels), ", "))
	}

	f.l = l
	return nil
}

func (f *LogLevelFlag) Type() string {
	return "format"
}

// Complete is the Cobra compatible completion function for this flag.
func (f *LogLevelFlag) Complete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return maps.Keys(levels), cobra.ShellCompDirectiveNoFileComp
}
