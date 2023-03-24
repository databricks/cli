package run

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type JobProgressEvent struct {
	Timestamp  time.Time
	JobId      int64
	RunId      int64
	RunName    string
	State      jobs.RunState
	RunPageURL string
}

func (event *JobProgressEvent) String() string {
	result := strings.Builder{}
	result.WriteString(event.Timestamp.Format("2006-01-02 15:04:05"))
	result.WriteString(" ")
	result.WriteString(event.RunName)
	result.WriteString(" ")
	result.WriteString(event.State.LifeCycleState.String())
	if event.State.ResultState.String() != "" {
		result.WriteString(" ")
		result.WriteString(event.State.ResultState.String())
	}
	result.WriteString(" ")
	result.WriteString(event.State.StateMessage)
	result.WriteString(" ")
	result.WriteString(event.RunPageURL)
	return result.String()
}

type JobProgressLogger struct {
	Mode      flags.ProgressLogFormat
	prevState *jobs.RunState
}

func NewJobProgressLogger(mode flags.ProgressLogFormat) *JobProgressLogger {
	return &JobProgressLogger{
		Mode: mode,
	}
}

func (l *JobProgressLogger) Log(event *JobProgressEvent) {
	if l.prevState != nil && l.prevState.LifeCycleState == event.State.LifeCycleState &&
		l.prevState.ResultState == event.State.ResultState {
		return
	}
	if l.prevState != nil && l.Mode == flags.ModeInplace {
		fmt.Fprint(os.Stderr, "\033[1F]")
	}
	if l.Mode == flags.ModeJson {
		b, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			// we panic because there we cannot catch this in json.RunNowAndWait
			panic(err)
		}
		fmt.Fprintln(os.Stderr, string(b))
	} else {
		fmt.Fprintln(os.Stderr, event.String())
	}
	l.prevState = &event.State
}
