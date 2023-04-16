package cmdio

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/databricks/bricks/libs/flags"
)

// This is the interface for all io interactions with a user
type Logger struct {
	Mode flags.ProgressLogFormat

	Reader bufio.Reader
	Writer io.Writer

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
		Mode:         flags.ModeJson,
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

func Ask(ctx context.Context, question string) (bool, error) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	return logger.Ask(question)
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
			// save cursor location
			l.Writer.Write([]byte("\033[s"))
		}

		// move cursor to saved location
		l.Writer.Write([]byte("\033[u"))

		// clear from cursor to end of screen
		l.Writer.Write([]byte("\033[0J"))

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

	default:
		// we panic because errors are not captured in some log sides like
		// jobs.RunNowAndWait
		panic("unknown progress logger mode: " + l.Mode.String())
	}
	l.isFirstEvent = false
}
