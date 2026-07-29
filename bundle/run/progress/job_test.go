package progress

import (
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestJobProgressEventString(t *testing.T) {
	event := &JobProgressEvent{
		Timestamp: time.Date(0, 0, 0, 0, 0, 0, 0, &time.Location{}),
		JobId:     123,
		RunId:     456,
		RunName:   "run_name",
		State: jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateSuccess,
			StateMessage:   "state_message",
		},
	}
	assert.Equal(t, "-0001-11-30 00:00:00 \"run_name\" TERMINATED SUCCESS state_message", event.String())
}

func TestJobStateTrackerPoll(t *testing.T) {
	running := &jobs.Run{RunId: 456, State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning}}
	terminated := &jobs.Run{RunId: 456, State: &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	}}

	var tracker JobStateTracker

	event, first := tracker.Poll(running)
	assert.True(t, first)
	assert.Equal(t, jobs.RunLifeCycleStateRunning, event.State.LifeCycleState)

	// The same state again is not worth reporting, and is no longer the first one.
	event, first = tracker.Poll(running)
	assert.Nil(t, event)
	assert.False(t, first)

	event, first = tracker.Poll(terminated)
	assert.False(t, first)
	assert.Equal(t, jobs.RunResultStateSuccess, event.State.ResultState)
}

func TestJobStateTrackerPollWithoutState(t *testing.T) {
	var tracker JobStateTracker

	event, first := tracker.Poll(&jobs.Run{RunId: 456})

	// A run reported without a state has no progress to report.
	assert.Nil(t, event)
	assert.False(t, first)
}
