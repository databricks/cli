package progress

import (
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJobProgressEventString(t *testing.T) {
	event := &JobProgressEvent{
		Timestamp: time.Date(0, 0, 0, 0, 0, 0, 0, &time.Location{}),
		JobId:     123,
		RunId:     456,
		RunName:   "run_name",
		State: jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleState_RunLifeCycleState_Terminated,
			ResultState:    jobs.RunResultState_RunResultState_Success,
			StateMessage:   new("state_message"),
		},
	}
	assert.Equal(t, "-0001-11-30 00:00:00 \"run_name\" TERMINATED SUCCESS state_message", event.String())
}
