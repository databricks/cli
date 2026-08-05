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
