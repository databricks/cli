package cmdio

import (
	"bufio"
	"encoding/json"
	"io"
	"os"

	"github.com/databricks/bricks/libs/flags"
)

type Logger struct {
	// Mode for the logger. One of (default, append, inplace, json).
	// default is resolved at runtime to one of (append, inplace)
	Mode flags.ProgressLogFormat

	// If true indicates inplace logging can be used if supported by the
	// command being run
	isInplaceSupported bool

	// Input stream (eg. stdin). Answers to questions prompted using the Ask() method
	// are read from here
	Reader bufio.Reader

	// Output stream where the logger writes to
	Writer io.Writer

	// If true, indicates no events have been printed by the logger yet. Used
	// by inplace logging for formatting
	isFirstEvent bool
}

func NewLogger(mode flags.ProgressLogFormat, isInplaceSupported bool) *Logger {
	return &Logger{
		Mode:               mode,
		Writer:             os.Stderr,
		Reader:             *bufio.NewReader(os.Stdin),
		isFirstEvent:       true,
		isInplaceSupported: isInplaceSupported,
	}
}

func (l *Logger) Ask(question string) (bool, error) {
	l.Writer.Write([]byte(question))
	ans, err := l.Reader.ReadString('\n')

	if err != nil {
		return false, err
	}

	if ans == "y\n" {
		return true, nil
	} else {
		return false, nil
	}
}

func (l *Logger) Log(event Event) {
	switch l.Mode {
	case flags.ModeInplace:
		if l.isFirstEvent {
			l.Writer.Write([]byte("\n"))
		}
		l.Writer.Write([]byte("\033[1F"))
		l.Writer.Write([]byte(event.String()))
		l.Writer.Write([]byte("\n"))

	case flags.ModeJson:
		b, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			// we panic because there we cannot catch this in jobs.RunNowAndWait
			panic(err)
		}
		l.Writer.Write([]byte(b))
		l.Writer.Write([]byte("\n"))

	case flags.ModeAppend:
		l.Writer.Write([]byte(event.String()))
		l.Writer.Write([]byte("\n"))

	// default to append format incase default mode was not resolved or an unexpected
	// mode is recieved. Ideally we should always explictly resolve default mode
	// at/before the call site
	default:
		l.Writer.Write([]byte(event.String()))
		l.Writer.Write([]byte("\n"))
	}
	l.isFirstEvent = false
}
