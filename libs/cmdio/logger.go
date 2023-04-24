package cmdio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/databricks/bricks/libs/flags"
)

// This is the interface for all io interactions with a user
type Logger struct {
	// Mode for the logger. One of (append, inplace, json).
	Mode flags.ProgressLogFormat

	// Input stream (eg. stdin). Answers to questions prompted using the Ask() method
	// are read from here
	Reader bufio.Reader

	// Output stream where the logger writes to
	Writer io.Writer

	// If true, indicates no events have been printed by the logger yet. Used
	// by inplace logging for formatting
	isFirstEvent bool
}

func NewLogger(mode flags.ProgressLogFormat) *Logger {
	return &Logger{
		Mode:         mode,
		Writer:       os.Stderr,
		Reader:       *bufio.NewReader(os.Stdin),
		isFirstEvent: true,
	}
}

func Default() *Logger {
	return &Logger{
		Mode:         flags.ModeAppend,
		Writer:       os.Stderr,
		Reader:       *bufio.NewReader(os.Stdin),
		isFirstEvent: true,
	}
}

func Log(ctx context.Context, event Event) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	logger.Log(event)
}

func LogString(ctx context.Context, message string) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	logger.Log(&MessageEvent{
		Message: message,
	})
}

func LogNewline(ctx context.Context) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	logger.Log(&NewlineEvent{})
}

func LogError(ctx context.Context, err error) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	logger.Log(&ErrorEvent{
		Error: err.Error(),
	})
}

func Ask(ctx context.Context, question string) (bool, error) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	return logger.Ask(question)
}

func (l *Logger) Ask(question string) (bool, error) {
	if l.Mode == flags.ModeJson {
		return false, fmt.Errorf("question prompts are not supported in json mode")
	}

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

func (l *Logger) writeJson(event Event) {
	b, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		// we panic because there we cannot catch this in jobs.RunNowAndWait
		panic(err)
	}
	l.Writer.Write([]byte(b))
	l.Writer.Write([]byte("\n"))
}

func (l *Logger) writeAppend(event Event) {
	l.Writer.Write([]byte(event.String()))
	l.Writer.Write([]byte("\n"))
}

func (l *Logger) writeInplace(event Event) {
	if l.isFirstEvent {
		// save cursor location
		l.Writer.Write([]byte("\033[s"))
	}

	// move cursor to saved location
	l.Writer.Write([]byte("\033[u"))

	// clear from cursor to end of screen
	l.Writer.Write([]byte("\033[0J"))

	l.Writer.Write([]byte(event.String()))
	l.Writer.Write([]byte("\n"))
	l.isFirstEvent = false
}

func (l *Logger) Log(event Event) {
	switch l.Mode {
	case flags.ModeInplace:
		if event.IsInplaceSupported() {
			l.writeInplace(event)
		} else {
			l.writeAppend(event)
		}

	case flags.ModeJson:
		l.writeJson(event)

	case flags.ModeAppend:
		l.writeAppend(event)

	default:
		// we panic because errors are not captured in some log sides like
		// jobs.RunNowAndWait
		panic("unknown progress logger mode: " + l.Mode.String())
	}

}
