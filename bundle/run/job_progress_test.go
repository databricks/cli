package run

import (
	"testing"
	"time"

	"github.com/databricks/bricks/libs/flags"
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
		RunPageURL: "run_url",
	}
	assert.Equal(t, "-0001-11-30 00:00:00 run_name TERMINATED SUCCESS state_message run_url", event.String())
}

func TestJobProgressEventLoggerErrorOnIncompatibleSettings(t *testing.T) {
	_, err := NewJobProgressLogger(flags.ModeInplace, "info", "stderr")
	assert.ErrorContains(t, err, "inplace progress logging cannot be used when log-file is stderr")
}

func TestInplaceJobsProgressLoggerCreatedWhenLoggingDisabled(t *testing.T) {
	_, err := NewJobProgressLogger(flags.ModeInplace, "disabled", "stderr")
	assert.NoError(t, err)
}

func TestInplaceJobsProgressLoggerCreatedWhenLogFileIsNotStderr(t *testing.T) {
	_, err := NewJobProgressLogger(flags.ModeInplace, "info", "stdout")
	assert.NoError(t, err)
}
