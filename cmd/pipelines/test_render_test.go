package pipelines

import (
	"bytes"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestTestProgressPrinter(t *testing.T) {
	var out bytes.Buffer
	p := &testProgressPrinter{out: &out}
	run := newTestRun()

	p.updateState(pipelines.UpdateInfoStateQueued)
	p.updateState(pipelines.UpdateInfoStateWaitingForResources)
	run.add(caseEvent(t, "t::timed", "RUNNING", "2026-01-01T00:00:00.000Z"))
	p.caseFinished(run.add(caseEvent(t, "t::timed", "PASSED", "2026-01-01T00:01:15.300Z")))
	p.updateState(pipelines.UpdateInfoStateRunning)
	p.caseFinished(run.add(caseEvent(t, "t::untimed", "SKIPPED", "2026-01-01T00:01:16.000Z")))

	assert.Equal(t, `Waiting for compute...
Running tests...
t::timed  PASSED (75.30s)
t::untimed  SKIPPED
`, out.String())
}

func TestRenderTestResult(t *testing.T) {
	tests := []struct {
		name   string
		events func() []testPipelineEvent
		state  pipelines.UpdateInfoState
		want   string
	}{
		{
			name: "failures",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{
					caseEvent(t, "t::ok", "PASSED", ""),
					caseEvent(t, "t::bad", "FAILED", "",
						`"error": {"exceptions": [{"class_name": "AssertionError", "message": "assert 1 == 2\nwhere 1 = f()"}]}`,
						`"truncation": {"truncated_fields": [{"field_name": "error"}]}`),
					caseEvent(t, "t::setup", "ERROR", ""),
					summaryEvent(t, 3, 1, 1, 0, 1),
				}
			},
			state: pipelines.UpdateInfoStateFailed,
			want: `
FAILURES
t::bad
  AssertionError: assert 1 == 2
  where 1 = f()
  (details truncated)
t::setup
  No details were reported.

3 tests: 1 passed, 1 failed, 0 skipped, 1 error
`,
		},
		{
			name: "several errors",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{
					caseEvent(t, "t::setup", "ERROR", ""),
					caseEvent(t, "t::teardown", "ERROR", ""),
					summaryEvent(t, 2, 0, 0, 0, 2),
				}
			},
			state: pipelines.UpdateInfoStateFailed,
			want: `
FAILURES
t::setup
  No details were reported.
t::teardown
  No details were reported.

2 tests: 0 passed, 0 failed, 0 skipped, 2 errors
`,
		},
		{
			name: "incomplete",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{caseEvent(t, "t::ok", "PASSED", ""), caseEvent(t, "t::skip", "SKIPPED", "")}
			},
			state: pipelines.UpdateInfoStateFailed,
			want: `
RUN INCOMPLETE: the update failed before all test results were received.
Results received: 1 passed, 1 skipped.
`,
		},
		{
			name: "incomplete with one error",
			events: func() []testPipelineEvent {
				return []testPipelineEvent{caseEvent(t, "t::setup", "ERROR", "")}
			},
			state: pipelines.UpdateInfoStateFailed,
			want: `
FAILURES
t::setup
  No details were reported.

RUN INCOMPLETE: the update failed before all test results were received.
Results received: 1 error.
`,
		},
		{
			name:   "incomplete without results",
			events: func() []testPipelineEvent { return nil },
			state:  pipelines.UpdateInfoStateCanceled,
			want: `
RUN INCOMPLETE: the update was canceled before all test results were received.
Results received: none.
`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := newTestRun()
			for _, e := range tc.events() {
				run.add(e)
			}
			var out bytes.Buffer
			renderTestResult(&out, run, run.incompleteReason(tc.state))
			assert.Equal(t, tc.want, out.String())
		})
	}
}
