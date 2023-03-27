package progress

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/bricks/libs/flags"
)

type Logger struct {
	Mode flags.ProgressLogFormat
}

func NewLogger(mode flags.ProgressLogFormat) *Logger {
	if mode == flags.ModeInplace {
		fmt.Fprintln(os.Stderr, "")
	}
	return &Logger{
		Mode: mode,
	}
}

func (l *Logger) Log(event Event) {
	switch l.Mode {
	case flags.ModeInplace:
		fmt.Fprint(os.Stderr, "\033[1F")
		fmt.Fprintln(os.Stderr, event.String())

	case flags.ModeJson:
		b, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			// we panic because there we cannot catch this in jobs.RunNowAndWait
			panic(err)
		}
		fmt.Fprintln(os.Stderr, string(b))

	case flags.ModeAppend:
		fmt.Fprintln(os.Stderr, event.String())

	default:
		// we panic because errors are not captured in some log sides like
		// jobs.RunNowAndWait
		panic("unknown progress logger mode: " + l.Mode.String())
	}
}
