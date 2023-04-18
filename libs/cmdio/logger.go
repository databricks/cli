package cmdio

import (
	"bufio"
	"encoding/json"
	"io"
	"os"

	"github.com/databricks/bricks/libs/flags"
)

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
