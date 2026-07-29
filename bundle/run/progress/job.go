package progress

import (
	"fmt"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type JobProgressEvent struct {
	Timestamp time.Time     `json:"timestamp"`
	JobId     int64         `json:"job_id"`
	RunId     int64         `json:"run_id"`
	RunName   string        `json:"run_name"`
	State     jobs.RunState `json:"state"`
}

// JobStateTracker turns the polls of a job run into one event per state change.
type JobStateTracker struct {
	prev *jobs.RunState
}

// Poll returns the event to report for this poll, or nil when the state has not
// changed. first is true for the state the run is seen in initially, where
// callers also report the run page URL.
func (t *JobStateTracker) Poll(run *jobs.Run) (event *JobProgressEvent, first bool) {
	if run.State == nil {
		return nil, false
	}
	first = t.prev == nil
	if !first && t.prev.LifeCycleState == run.State.LifeCycleState && t.prev.ResultState == run.State.ResultState {
		return nil, false
	}
	t.prev = run.State
	return &JobProgressEvent{
		Timestamp: time.Now(),
		JobId:     run.JobId,
		RunId:     run.RunId,
		RunName:   run.RunName,
		State:     *run.State,
	}, first
}

func (event *JobProgressEvent) String() string {
	result := strings.Builder{}
	result.WriteString(event.Timestamp.Format("2006-01-02 15:04:05") + " ")
	result.WriteString(fmt.Sprintf(`"%s"`, event.RunName) + " ")
	result.WriteString(event.State.LifeCycleState.String())

	resultState := event.State.ResultState.String()
	if resultState != "" {
		result.WriteString(" " + resultState)
	}

	stateMessage := event.State.StateMessage
	if stateMessage != "" {
		result.WriteString(" " + stateMessage)
	}

	return result.String()
}
