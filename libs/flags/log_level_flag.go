package flags

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

var levels = map[string]slog.Level{
	"trace":    log.LevelTrace,
	"debug":    log.LevelDebug,
	"info":     log.LevelInfo,
	"warn":     log.LevelWarn,
	"error":    log.LevelError,
	"disabled": log.LevelDisabled,
}

type LogLevelFlag struct {
	l slog.Level
}

func NewLogLevelFlag() LogLevelFlag {
	return LogLevelFlag{
		l: log.LevelWarn,
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
