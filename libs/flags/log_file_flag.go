package flags

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type LogFileFlag struct {
	s string
	f *os.File
	w io.Writer
}

func NewLogFileFlag() LogFileFlag {
	return LogFileFlag{
		s: "stderr",
		w: os.Stderr,
	}
}

func (f *LogFileFlag) Writer() io.Writer {
	return f.w
}

func (f *LogFileFlag) Close() error {
	if f.f == nil {
		return nil
	}
	return f.f.Close()
}

func (f *LogFileFlag) String() string {
	return f.s
}

func (f *LogFileFlag) Set(s string) error {
	lower := strings.ToLower(s)
	switch lower {
	case "stderr":
		f.w = os.Stderr
		f.s = lower
	case "stdout":
		f.w = os.Stdout
		f.s = lower
	default:
		file, err := os.OpenFile(s, os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return err
		}

		f.f = file
		f.w = file
		f.s = s
	}

	return nil
}

func (f *LogFileFlag) Type() string {
	return "file"
}

// Complete is the Cobra compatible completion function for this flag.
func (f *LogFileFlag) Complete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"stdout",
		"stderr",
	}, cobra.ShellCompDirectiveDefault
}
