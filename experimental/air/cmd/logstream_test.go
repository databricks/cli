package aircmd

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestClassifyLogError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error // errBricklensFeatureDisabled, the input error, or nil
	}{
		{
			name: "feature disabled falls back",
			err:  &apierr.APIError{ErrorCode: "FEATURE_DISABLED", StatusCode: http.StatusForbidden},
			want: errBricklensFeatureDisabled,
		},
		{
			name: "endpoint not found falls back",
			err:  &apierr.APIError{ErrorCode: "ENDPOINT_NOT_FOUND", StatusCode: http.StatusNotFound},
			want: errBricklensFeatureDisabled,
		},
		{
			name: "bare 404 falls back",
			err:  &apierr.APIError{ErrorCode: "SOMETHING", StatusCode: http.StatusNotFound},
			want: errBricklensFeatureDisabled,
		},
		{
			name: "genuine resource-does-not-exist surfaces",
			err:  apierr.ErrResourceDoesNotExist,
			want: apierr.ErrResourceDoesNotExist,
		},
		{
			name: "transient 500 is retried",
			err:  &apierr.APIError{ErrorCode: "INTERNAL", StatusCode: http.StatusInternalServerError},
			want: nil,
		},
		{
			name: "plain error is retried",
			err:  errors.New("connection reset"),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLogError(tt.err)
			switch tt.want {
			case errBricklensFeatureDisabled:
				assert.ErrorIs(t, got, errBricklensFeatureDisabled)
			case nil:
				assert.NoError(t, got)
			default:
				assert.ErrorIs(t, got, tt.want)
			}
		})
	}
}

func TestProjectRunStatus(t *testing.T) {
	run := &jobs.Run{
		StartTime: 1000,
		EndTime:   2000,
		State: &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateSuccess,
			StateMessage:   "done",
		},
		Tasks: []jobs.RunTask{
			{AttemptNumber: 0},
			{AttemptNumber: 2},
			{AttemptNumber: 1},
		},
	}

	s := projectRunStatus(run)
	assert.Equal(t, "TERMINATED", s.lifeCycleState)
	assert.Equal(t, "SUCCESS", s.resultState)
	assert.Equal(t, "done", s.stateMessage)
	assert.Equal(t, int64(1000), s.startTimeMs)
	assert.Equal(t, int64(2000), s.endTimeMs)
	assert.Equal(t, 2, s.latestAttempt)
	assert.True(t, s.terminal())
	assert.True(t, s.succeeded())
	assert.Equal(t, "SUCCESS", s.displayState())
}

func TestLogRunStatusTerminal(t *testing.T) {
	tests := []struct {
		name         string
		lifeCycle    string
		resultState  string
		wantTerminal bool
	}{
		{"running", "RUNNING", "", false},
		{"pending", "PENDING", "", false},
		{"terminated lifecycle", "TERMINATED", "", true},
		{"internal error lifecycle", "INTERNAL_ERROR", "", true},
		{"failed result", "TERMINATING", "FAILED", true},
		{"canceled result", "RUNNING", "CANCELED", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := logRunStatus{lifeCycleState: tt.lifeCycle, resultState: tt.resultState}
			assert.Equal(t, tt.wantTerminal, s.terminal())
		})
	}
}

func TestLogRequestFromSeconds(t *testing.T) {
	now := time.Unix(10_000, 0)

	// --minutes narrows the window to now - N*60.
	req := logRequest{windowMinutes: 5}
	assert.Equal(t, int64(10_000-300), req.fromSeconds(logRunStatus{startTimeMs: 1_000_000}, now))

	// No window: from the run's start second.
	req = logRequest{}
	assert.Equal(t, int64(1000), req.fromSeconds(logRunStatus{startTimeMs: 1_000_000}, now))

	// No window, run not started: everything stored (0).
	assert.Equal(t, int64(0), req.fromSeconds(logRunStatus{}, now))
}

func TestLogRequestToSeconds(t *testing.T) {
	req := logRequest{}

	// Active run: 0 lets the endpoint default to now.
	assert.Equal(t, int64(0), req.toSeconds(logRunStatus{lifeCycleState: "RUNNING"}))

	// Terminal run: ceil of the end millisecond so the final partial second is kept.
	terminal := logRunStatus{lifeCycleState: "TERMINATED", resultState: "SUCCESS", endTimeMs: 2001}
	assert.Equal(t, int64(3), req.toSeconds(terminal))
}

func TestLogRequestTailTarget(t *testing.T) {
	assert.Equal(t, defaultCompletedRunTailLines, logRequest{}.tailTarget())
	assert.Equal(t, 42, logRequest{tailLines: 42}.tailTarget())
}

func TestSeenSetEviction(t *testing.T) {
	s := newSeenSet(2)
	s.add(1, "a")
	s.add(2, "b")
	assert.True(t, s.has(1, "a"))
	assert.True(t, s.has(2, "b"))

	// Adding a third evicts the oldest-inserted (1,"a").
	s.add(3, "c")
	assert.False(t, s.has(1, "a"))
	assert.True(t, s.has(2, "b"))
	assert.True(t, s.has(3, "c"))

	// Same (nano, body) shares one entry; distinct body under the same nano does not.
	s.add(3, "c")
	assert.True(t, s.has(3, "c"))
	assert.False(t, s.has(3, "d"))
}
