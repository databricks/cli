package dresources

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jobRunClientFor returns a client talking to server. Call it after the test
// registers its own handlers: first registration wins, so the defaults added here
// only fill the gaps.
func jobRunClientFor(t *testing.T, server *testserver.Server) *databricks.WorkspaceClient {
	t.Helper()
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)
	return client
}

// jobRunServer returns a client whose runs/get is the given handler, so a wait can
// be driven without a real run.
func jobRunServer(t *testing.T, getRun testserver.HandlerFunc) *databricks.WorkspaceClient {
	t.Helper()
	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", getRun)
	return jobRunClientFor(t, server)
}

// The Jobs API reports the run page in the legacy fragment form; errors and
// progress lines carry the path form it converts to.
const (
	testRunPageURL  = "https://myworkspace.databricks.test/?o=900800700600#job/456/run/123"
	testRunPageLink = "run page: https://myworkspace.databricks.test/jobs/456/runs/123?o=900800700600"
)

// jobRunClient returns a client whose GetRun always reports the given run state.
func jobRunClient(t *testing.T, state *jobs.RunState) *databricks.WorkspaceClient {
	t.Helper()
	return jobRunServer(t, func(req testserver.Request) any {
		return jobs.Run{RunId: 123, JobId: 456, State: state, RunPageUrl: testRunPageURL}
	})
}

// waitForTestRun drives the framework hook, so it covers parsing the id the
// framework hands back from DoCreate along with the wait itself.
func waitForTestRun(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient) (*JobRunRemote, error) {
	t.Helper()
	r := (&ResourceJobRun{}).New(client)
	return r.WaitAfterCreate(ctx, "123", &JobRunState{})
}

func TestJobRunWaitFailsOnFailedResult(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateFailed,
		StateMessage:   "task failed",
	})

	_, err := waitForTestRun(t, t.Context(), client)

	require.ErrorContains(t, err, "did not succeed: FAILED: task failed")
}

// Without the deprecated per-task state, a failed task is told apart from a
// skipped one by its termination details.
func TestJobRunWaitReportsFailedTaskWithoutDeprecatedState(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		return jobs.Run{
			RunId: 123,
			JobId: 456,
			State: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminated,
				ResultState:    jobs.RunResultStateFailed,
			},
			Tasks: []jobs.RunTask{
				{TaskKey: "main", RunId: 999, Status: &jobs.RunStatus{
					State: jobs.RunLifecycleStateV2StateTerminated,
					TerminationDetails: &jobs.TerminationDetails{
						Type:    jobs.TerminationTypeTypeClientError,
						Code:    jobs.TerminationCodeCodeRunExecutionError,
						Message: "Workload failed, see run output for details",
					},
				}},
				// Never ran, so it has no error of its own to report.
				{TaskKey: "downstream", RunId: 1000, Status: &jobs.RunStatus{
					State: jobs.RunLifecycleStateV2StateTerminated,
					TerminationDetails: &jobs.TerminationDetails{
						Type: jobs.TerminationTypeTypeClientError,
						Code: jobs.TerminationCodeCodeSkipped,
					},
				}},
			},
		}
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		return jobs.RunOutput{Error: "RuntimeError: intentional failure"}
	})

	_, err := waitForTestRun(t, t.Context(), jobRunClientFor(t, server))

	require.ErrorContains(t, err, `task "main": RuntimeError: intentional failure`)
	assert.NotContains(t, err.Error(), `task "downstream"`)
}

// When the run output carries no error, the termination message stands in.
func TestJobRunWaitFallsBackToTheTerminationMessage(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		return jobs.Run{
			RunId: 123,
			JobId: 456,
			State: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminated,
				ResultState:    jobs.RunResultStateFailed,
			},
			Tasks: []jobs.RunTask{
				{TaskKey: "main", RunId: 999, Status: &jobs.RunStatus{
					State: jobs.RunLifecycleStateV2StateTerminated,
					TerminationDetails: &jobs.TerminationDetails{
						Type:    jobs.TerminationTypeTypeCloudFailure,
						Code:    jobs.TerminationCodeCodeCloudFailure,
						Message: "the cloud provider ran out of capacity",
					},
				}},
			},
		}
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		return jobs.RunOutput{}
	})

	_, err := waitForTestRun(t, t.Context(), jobRunClientFor(t, server))

	require.ErrorContains(t, err, `task "main": the cloud provider ran out of capacity`)
}

// The same fallback for a task still reported through the deprecated state.
func TestJobRunWaitFallsBackToTheTaskMessage(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		return jobs.Run{
			RunId: 123,
			JobId: 456,
			State: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminated,
				ResultState:    jobs.RunResultStateFailed,
			},
			Tasks: []jobs.RunTask{
				{TaskKey: "main", RunId: 999, State: &jobs.RunState{
					LifeCycleState: jobs.RunLifeCycleStateTerminated,
					ResultState:    jobs.RunResultStateFailed,
					StateMessage:   "Workload failed, see run output for details",
				}},
			},
		}
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		return jobs.RunOutput{}
	})

	_, err := waitForTestRun(t, t.Context(), jobRunClientFor(t, server))

	require.ErrorContains(t, err, `task "main": Workload failed, see run output for details`)
}

func TestJobRunWaitFailsOnSkipped(t *testing.T) {
	// A skipped run has no result_state, so the lifecycle state is reported.
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateSkipped,
	})

	_, err := waitForTestRun(t, t.Context(), client)

	require.ErrorContains(t, err, "did not succeed: SKIPPED")
}

func TestJobRunWaitFailsOnInternalError(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateInternalError,
	})

	_, err := waitForTestRun(t, t.Context(), client)

	require.ErrorContains(t, err, "run did not succeed: INTERNAL_ERROR")
	require.ErrorContains(t, err, testRunPageLink)
}

func TestJobRunWaitReportsOnlyTheLastAttemptOfATask(t *testing.T) {
	failed := &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateFailed,
	}
	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		return jobs.Run{
			RunId: 123,
			JobId: 456,
			State: failed,
			Tasks: []jobs.RunTask{
				{TaskKey: "main", RunId: 998, AttemptNumber: 0, State: failed},
				{TaskKey: "main", RunId: 999, AttemptNumber: 1, State: failed},
			},
		}
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		return jobs.RunOutput{Error: "output of run " + req.URL.Query().Get("run_id")}
	})

	_, err := waitForTestRun(t, t.Context(), jobRunClientFor(t, server))

	// The Jobs API reports a retried task once per attempt; only the last one says
	// how the run ended up.
	require.ErrorContains(t, err, `task "main": output of run 999`)
	assert.NotContains(t, err.Error(), "output of run 998")
}

func TestJobRunWaitAbandonedLinksTheRun(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning})

	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()

	_, err := waitForTestRun(t, ctx, client)

	// The run keeps going, so the error links to it and names the interrupt.
	require.ErrorContains(t, err, "interrupted while waiting for the run to finish")
	require.ErrorContains(t, err, testRunPageLink)
}

// State written before lifecycle existed has no such key, and must still load.
func TestJobRunStateUnmarshalWithoutLifecycle(t *testing.T) {
	var state JobRunState

	require.NoError(t, json.Unmarshal([]byte(`{}`), &state))

	assert.Nil(t, state.Lifecycle.Triggers.OnFileChange)
}

// The planner diffs RemapState(remote) against PrepareState(config), so a run
// that did not end in SUCCESS has to surface as a difference on result_state.
func TestJobRunRemapStateCarriesTheOutcome(t *testing.T) {
	for _, outcome := range []jobs.RunResultState{
		jobs.RunResultStateSuccess,
		jobs.RunResultStateFailed,
		// A run still going, and a SKIPPED one, report no result at all.
		"",
	} {
		t.Run(string(outcome), func(t *testing.T) {
			remote := &JobRunRemote{RunId: 123, ResultState: outcome}

			state := (&ResourceJobRun{}).RemapState(remote)

			assert.Equal(t, outcome, state.ResultState)
			assert.Equal(t, emptyJobRunLifecycleState(), state.Lifecycle)
		})
	}
}

// resources.yml ignores remote drift on everything the RunNow request carries,
// since GetRun does not echo it back faithfully, and leaves result_state alone.
func TestJobRunIgnoresEveryRequestField(t *testing.T) {
	adapters, err := InitAll(nil)
	require.NoError(t, err)
	ignored := adapters["job_runs"].ResourceConfig().IgnoreRemoteChanges

	for field := range reflect.TypeFor[jobs.RunNow]().Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		assert.True(t, ignoresRemoteChanges(ignored, name), "jobs.RunNow field %q is not in job_runs ignore_remote_changes", name)
	}

	assert.False(t, ignoresRemoteChanges(ignored, "result_state"), "result_state must stay comparable against the remote")
}

// ignoresRemoteChanges reports whether the rules suppress remote drift on field.
func ignoresRemoteChanges(rules []FieldRule, field string) bool {
	path := structpath.MustParsePath(field)
	return slices.ContainsFunc(rules, func(r FieldRule) bool { return path.HasPatternPrefix(r.Field) })
}

// Reporting RUNNING for the first two polls exercises the poll loop.
func TestJobRunWaitPollsUntilTerminal(t *testing.T) {
	var gets atomic.Int32
	client := jobRunServer(t, func(req testserver.Request) any {
		if gets.Add(1) <= 2 {
			return jobs.Run{RunId: 123, JobId: 456, State: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateRunning,
			}}
		}
		return jobs.Run{RunId: 123, JobId: 456, State: &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateSuccess,
		}}
	})

	remote, err := waitForTestRun(t, t.Context(), client)
	require.NoError(t, err)

	// SUCCESS is only reachable by polling past the RUNNING reads.
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
	assert.Equal(t, int32(3), gets.Load(), "expected the wait to poll past both RUNNING reads")
}

func TestJobRunCreateSendsAFreshIdempotencyToken(t *testing.T) {
	var tokens []string
	server := testserver.New(t)
	server.Handle("POST", "/api/2.2/jobs/run-now", func(req testserver.Request) any {
		var body jobs.RunNow
		require.NoError(t, json.Unmarshal(req.Body, &body))
		tokens = append(tokens, body.IdempotencyToken)
		return jobs.RunNowResponse{RunId: int64(123 + len(tokens))}
	})
	r := (&ResourceJobRun{}).New(jobRunClientFor(t, server))
	config := &JobRunState{RunNow: jobs.RunNow{JobId: 456}}

	for range 2 {
		_, _, err := r.DoCreate(t.Context(), config)
		require.NoError(t, err)
	}

	require.Len(t, tokens, 2)
	assert.NotEmpty(t, tokens[0])
	assert.NotEqual(t, tokens[0], tokens[1])
	// Token must not leak into persisted state.
	assert.Empty(t, config.IdempotencyToken)
}

func TestCompactJobRunValue(t *testing.T) {
	assert.Equal(t, strings.Repeat("a", jobRunValueHashLength), compactJobRunValue(strings.Repeat("a", jobRunValueHashLength)))
	assert.Equal(
		t,
		"sha256:d66304b6180365e47c858f6c84d3da065caf4b3350c9f45277a1af82e3dbb055",
		compactJobRunValue(strings.Repeat("a", jobRunValueHashLength+1)),
	)
}

func TestJobRunValueChangeStateNormalizesAfterAllReferencesResolve(t *testing.T) {
	expr := "${resources.jobs.other.id}-${resources.jobs.extra.id}"
	var trigger resources.JobRunTrigger
	trigger.OnValueChange = &expr
	var input resources.JobRun
	input.Lifecycle = &resources.JobRunLifecycle{}
	input.Lifecycle.Triggers = []resources.JobRunTrigger{trigger}
	state := (&ResourceJobRun{}).PrepareState(&input)
	path := structpath.NewStringKey(structpath.MustParsePath("lifecycle.triggers.on_value_change"), expr)
	sv := structvar.NewStructVar(state, map[string]string{path.String(): expr})

	require.NoError(t, sv.ResolveRef("${resources.jobs.other.id}", int64(123)))
	assert.Equal(t, map[string]string{expr: "123-${resources.jobs.extra.id}"}, state.Lifecycle.Triggers.OnValueChange)

	require.NoError(t, sv.ResolveRef("${resources.jobs.extra.id}", int64(456)))
	assert.Equal(t, map[string]string{"123-456": "123-456"}, state.Lifecycle.Triggers.OnValueChange)
}

func TestJobRunDeleteLeavesFinishedRunAlone(t *testing.T) {
	var cancelled atomic.Bool
	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		return jobs.Run{RunId: 123, JobId: 456, State: &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateSuccess,
		}}
	})
	server.Handle("POST", "/api/2.2/jobs/runs/cancel", func(req testserver.Request) any {
		cancelled.Store(true)
		return testserver.Response{}
	})
	server.Handle("POST", "/api/2.2/jobs/runs/delete", func(req testserver.Request) any {
		return testserver.Response{}
	})
	r := (&ResourceJobRun{}).New(jobRunClientFor(t, server))

	require.NoError(t, r.DoDelete(t.Context(), "123", &JobRunState{}))

	assert.False(t, cancelled.Load(), "a run that already finished has nothing to cancel")
}

func TestJobRunOverrideChangeDescTriggerRemoved(t *testing.T) {
	r := &ResourceJobRun{}
	for _, tt := range []struct {
		name   string
		path   string
		old    any
		new    any
		action deployplan.ActionType
	}{
		{"cleared on_bundle_deploy string", "lifecycle.triggers.on_bundle_deploy", "old", "", deployplan.Skip},
		{"nil on_bundle_deploy", "lifecycle.triggers.on_bundle_deploy", "old", nil, deployplan.Skip},
		{"rotated on_bundle_deploy", "lifecycle.triggers.on_bundle_deploy", "old", "uuid", deployplan.Recreate},
		{"cleared on_file_change", "lifecycle.triggers.on_file_change", map[string]string{"a.txt": "h"}, nil, deployplan.Skip},
		{"changed on_file_change map", "lifecycle.triggers.on_file_change", map[string]string{"a.txt": "h"}, map[string]string{"a.txt": "new"}, deployplan.Recreate},
		// A file dropping out of the map is a real change, so the skip must not
		// extend to on_file_change entries.
		{"removed one on_file_change", "lifecycle.triggers.on_file_change", map[string]string{"a.txt": "h", "b.txt": "h"}, map[string]string{"a.txt": "h"}, deployplan.Recreate},
		{"cleared on_file_change child", "lifecycle.triggers.on_file_change['a.txt']", "h", nil, deployplan.Recreate},
		{"cleared on_value_change", "lifecycle.triggers.on_value_change", map[string]string{"a": "a"}, nil, deployplan.Skip},
		{"removed one on_value_change", "lifecycle.triggers.on_value_change", map[string]string{"a": "a", "b": "b"}, map[string]string{"b": "b"}, deployplan.Skip},
		{"changed on_value_change", "lifecycle.triggers.on_value_change", map[string]string{"a": "a"}, map[string]string{"b": "b"}, deployplan.Recreate},
		{"cleared on_value_change child", "lifecycle.triggers.on_value_change['b']", "b", nil, deployplan.Skip},
		{"result_state with unreadable remote", "result_state", "", nil, deployplan.Recreate},
	} {
		t.Run(tt.name, func(t *testing.T) {
			change := &ChangeDesc{Action: deployplan.Recreate, Old: tt.old, New: tt.new}
			require.NoError(t, r.OverrideChangeDesc(t.Context(), structpath.MustParsePath(tt.path), change, nil))
			assert.Equal(t, tt.action, change.Action)
		})
	}
}
