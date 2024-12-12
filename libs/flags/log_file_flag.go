package flags

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Abstract over files that are already open (e.g. stderr) and
// files that need to be opened before use.
type logFile interface {
	Writer() io.Writer
	Open() error
	Close() error
}

// nopLogFile implements [logFile] for [os.Stderr] and [os.Stdout].
// The [logFile.Open] and [logFile.Close] functions do nothing.
type nopLogFile struct {
	f *os.File
}

func (f *nopLogFile) Writer() io.Writer {
	return f.f
}

func (f *nopLogFile) Open() error {
	return nil
}

func (f *nopLogFile) Close() error {
	return nil
}

// nopLogFile implements [logFile] for actual files.
type realLogFile struct {
	s string
	f *os.File
}

func (f *realLogFile) Writer() io.Writer {
	if f.f == nil {
		panic("file hasn't been opened")
	}
	return f.f
}

func (f *realLogFile) Open() error {
	file, err := os.OpenFile(f.s, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}

	f.f = file
	return nil
}

func (f *realLogFile) Close() error {
	if f.f == nil {
		return nil
	}
	return f.f.Close()
}

type LogFileFlag struct {
	name string
	logFile
}

func NewLogFileFlag() LogFileFlag {
	return LogFileFlag{
		name:    "stderr",
		logFile: &nopLogFile{os.Stderr},
	}
}

func (f *LogFileFlag) String() string {
	return f.name
}

func (f *LogFileFlag) Set(s string) error {
	lower := strings.ToLower(s)
	switch lower {
	case "stderr":
		f.name = lower
		f.logFile = &nopLogFile{os.Stderr}
	case "stdout":
		f.name = lower
		f.logFile = &nopLogFile{os.Stdout}
	default:
		f.name = s
		f.logFile = &realLogFile{s: s}
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
