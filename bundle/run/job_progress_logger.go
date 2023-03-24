package run

import (
	"fmt"
	"os"

	"github.com/databricks/bricks/libs/progress"
)

type TextLoggerMode string

var ModeAppend = TextLoggerMode("append")
var ModeInplace = TextLoggerMode("inplace")

type JobTextLogger struct {
	Mode TextLoggerMode

	// maps event Ids to runId for inplace logging
	eventIdMemo     map[int64]int
	Renderer *progress.EventRenderer
}

func NewJobTextLogger(mode TextLoggerMode) *JobTextLogger {
	switch mode {
	case ModeAppend:
		return &JobTextLogger{Mode: mode}
	case ModeInplace:
		return &JobTextLogger{
			Mode:     mode,
			eventIdMemo:     make(map[int64]int),
			Renderer: progress.NewEventRenderer(),
		}
	}
	return nil
}

func (l *JobTextLogger) Log(event *JobProgressEvent) {
	switch l.Mode {
	case ModeAppend:
		fmt.Fprintf(os.Stderr, "%s %s", event.Timestamp, event.Content())
	case ModeInplace:
		eventId, ok := l.eventIdMemo[event.RunId]
		if !ok {
			id := l.Renderer.AddEvent(event.EventState(), event.Content(), event.IndentLevel())
			l.eventIdMemo[event.RunId] = id
			return
		}
		l.Renderer.UpdateState(eventId, event.EventState())
		l.Renderer.UpdateContent(eventId, event.Content())
	}
}
