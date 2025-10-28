package cmdio

import (
	"bufio"
	"io"
	"os"

	"github.com/databricks/cli/libs/flags"
)

// This is the interface for all io interactions with a user
type Logger struct {
	// Mode for the logger. One of (append).
	Mode flags.ProgressLogFormat

	// Input stream (eg. stdin). Answers to questions prompted using the Ask() method
	// are read from here
	Reader bufio.Reader

	// Output stream where the logger writes to
	Writer io.Writer
}

func NewLogger(mode flags.ProgressLogFormat) *Logger {
	return &Logger{
		Mode:   mode,
		Writer: os.Stderr,
		Reader: *bufio.NewReader(os.Stdin),
	}
}

func Default() *Logger {
	return &Logger{
		Mode:   flags.ModeAppend,
		Writer: os.Stderr,
		Reader: *bufio.NewReader(os.Stdin),
	}
}
