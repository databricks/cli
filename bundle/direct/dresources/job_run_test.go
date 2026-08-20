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
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/structs/structpath"
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

func TestJobRunWaitSucceeds(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	})

	remote, err := waitForTestRun(t, t.Context(), client)

	require.NoError(t, err)
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
}

func TestReportRunLineIncludesResourceKey(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = WithResourceKey(ctx, "job_runs.my_run")

	reportRunLine(ctx, 123, "SUCCESS")

	assert.Equal(t, "Output from job_runs.my_run: id=123: SUCCESS\n", stderr.String())
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

func TestJobRunWaitReportsFailedTask(t *testing.T) {
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
				{TaskKey: "ok", RunId: 998, State: &jobs.RunState{
					LifeCycleState: jobs.RunLifeCycleStateTerminated,
					ResultState:    jobs.RunResultStateSuccess,
				}},
				{TaskKey: "main", RunId: 999, State: failed},
			},
		}
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		return jobs.RunOutput{Error: "notebook not found"}
	})

	_, err := waitForTestRun(t, t.Context(), jobRunClientFor(t, server))

	require.ErrorContains(t, err, `task "main": notebook not found`)
	assert.NotContains(t, err.Error(), `task "ok"`)
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

// A real workspace reports a run whose task failed as INTERNAL_ERROR in the
// deprecated life_cycle_state. The failing task still has to be named.
func TestJobRunWaitReportsFailedTaskOfInternalErrorRun(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		return jobs.Run{
			RunId:      123,
			JobId:      456,
			RunPageUrl: testRunPageURL,
			State: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateInternalError,
				ResultState:    jobs.RunResultStateFailed,
				StateMessage:   "Task main failed with message: Workload failed, see run output for details.",
			},
			Tasks: []jobs.RunTask{
				{TaskKey: "main", RunId: 999, State: &jobs.RunState{
					LifeCycleState: jobs.RunLifeCycleStateTerminated,
					ResultState:    jobs.RunResultStateFailed,
				}},
			},
		}
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		return jobs.RunOutput{Error: "RuntimeError: intentional failure"}
	})

	_, err := waitForTestRun(t, t.Context(), jobRunClientFor(t, server))

	require.ErrorContains(t, err, "run did not succeed: FAILED")
	require.ErrorContains(t, err, `task "main": RuntimeError: intentional failure`)
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

// An abandoned wait leaves the run going with its id recorded, so the next deploy
// reads an empty outcome, which result_state drift catches.
func TestJobRunReadOfUnfinishedRunReportsNoResult(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning})

	remote, err := (&ResourceJobRun{}).New(client).DoRead(t.Context(), "123")

	require.NoError(t, err)
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunLifeCycleStateRunning, remote.State.LifeCycleState)
	assert.Empty(t, remote.ResultState)
}

// PrepareState records the outcome the run must reach, the same for every run,
// so the planner has something to compare the remote against.
func TestJobRunPrepareStateRequiresSuccess(t *testing.T) {
	state := (&ResourceJobRun{}).PrepareState(&resources.JobRun{RunNow: jobs.RunNow{JobId: 456}})

	assert.Equal(t, jobs.RunResultStateSuccess, state.ResultState)
}

func TestJobRunPrepareStateOnBundleDeploy(t *testing.T) {
	on := true
	input := &resources.JobRun{
		Lifecycle: &resources.JobRunLifecycle{
			Triggers: []resources.JobRunTrigger{{OnBundleDeploy: &on}},
		},
	}
	first := (&ResourceJobRun{}).PrepareState(input)
	require.NotNil(t, first.Lifecycle)
	require.NotNil(t, first.Lifecycle.Triggers)
	assert.NotEmpty(t, first.Lifecycle.Triggers.OnBundleDeploy)

	second := (&ResourceJobRun{}).PrepareState(input)
	assert.NotEqual(t, first.Lifecycle.Triggers.OnBundleDeploy, second.Lifecycle.Triggers.OnBundleDeploy)
}

func TestJobRunPrepareStateOnFileChange(t *testing.T) {
	hashes := map[string]string{"a.txt": "abc"}

	t.Run("armed", func(t *testing.T) {
		state := (&ResourceJobRun{}).PrepareState(&resources.JobRun{
			ResolvedFileTriggers: hashes,
		})
		require.NotNil(t, state.Lifecycle)
		require.NotNil(t, state.Lifecycle.Triggers)
		assert.Equal(t, hashes, state.Lifecycle.Triggers.OnFileChange)
		assert.Empty(t, state.Lifecycle.Triggers.OnBundleDeploy)
	})

	t.Run("both triggers", func(t *testing.T) {
		on := true
		state := (&ResourceJobRun{}).PrepareState(&resources.JobRun{
			Lifecycle: &resources.JobRunLifecycle{
				Triggers: []resources.JobRunTrigger{{OnBundleDeploy: &on}},
			},
			ResolvedFileTriggers: hashes,
		})
		require.NotNil(t, state.Lifecycle)
		require.NotNil(t, state.Lifecycle.Triggers)
		assert.NotEmpty(t, state.Lifecycle.Triggers.OnBundleDeploy)
		assert.Equal(t, hashes, state.Lifecycle.Triggers.OnFileChange)
	})
}

func TestJobRunOverrideChangeDescTriggerRemoved(t *testing.T) {
	r := &ResourceJobRun{}

	t.Run("clearing lifecycle downgrades to skip", func(t *testing.T) {
		change := &ChangeDesc{
			Action: deployplan.Recreate,
			Old:    &JobRunLifecycleState{Triggers: &JobRunTriggersState{OnBundleDeploy: "old"}},
			New:    nil,
		}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), structpath.MustParsePath("lifecycle"), change, nil))
		assert.Equal(t, deployplan.Skip, change.Action)
		assert.Equal(t, "trigger removed", change.Reason)
	})

	t.Run("clearing on_bundle_deploy leaf downgrades to skip", func(t *testing.T) {
		change := &ChangeDesc{
			Action: deployplan.Recreate,
			Old:    "old",
			New:    "",
		}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), structpath.MustParsePath("lifecycle.triggers.on_bundle_deploy"), change, nil))
		assert.Equal(t, deployplan.Skip, change.Action)
		assert.Equal(t, "trigger removed", change.Reason)
	})

	t.Run("clearing on_file_change leaf downgrades to update", func(t *testing.T) {
		change := &ChangeDesc{
			Action: deployplan.Recreate,
			Old:    map[string]string{"a.txt": "abc"},
			New:    nil,
		}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), structpath.MustParsePath("lifecycle.triggers.on_file_change"), change, nil))
		assert.Equal(t, deployplan.Update, change.Action)
		assert.Equal(t, "trigger removed", change.Reason)
	})

	t.Run("fresh fingerprint still recreates", func(t *testing.T) {
		change := &ChangeDesc{
			Action: deployplan.Recreate,
			Old:    "old",
			New:    "new",
		}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), structpath.MustParsePath("lifecycle.triggers.on_bundle_deploy"), change, nil))
		assert.Equal(t, deployplan.Recreate, change.Action)
	})

	t.Run("changed on_file_change hash still recreates", func(t *testing.T) {
		change := &ChangeDesc{
			Action: deployplan.Recreate,
			Old:    map[string]string{"a.txt": "old"},
			New:    map[string]string{"a.txt": "new"},
		}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), structpath.MustParsePath("lifecycle.triggers.on_file_change"), change, nil))
		assert.Equal(t, deployplan.Recreate, change.Action)
	})
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

// jobRunDeletion records what the fake workspace saw while a run was deleted.
type jobRunDeletion struct {
	cancelled       atomic.Bool
	settled         atomic.Bool
	settledAtDelete atomic.Bool
}

// jobRunDeleteClient returns a client for a run in the given state, whose cancel
// settles one poll late the way the API's asynchronous cancellation does.
func jobRunDeleteClient(t *testing.T, state *jobs.RunState) (*databricks.WorkspaceClient, *jobRunDeletion) {
	t.Helper()
	var deletion jobRunDeletion
	cancelled := &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateCanceled,
	}

	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		current := state
		switch {
		case deletion.settled.Load():
			current = cancelled
		case deletion.cancelled.Load():
			// Report the run's old state once more, then settle on the next poll.
			deletion.settled.Store(true)
		}
		return jobs.Run{RunId: 123, JobId: 456, State: current}
	})
	server.Handle("POST", "/api/2.2/jobs/runs/cancel", func(req testserver.Request) any {
		deletion.cancelled.Store(true)
		return testserver.Response{}
	})
	server.Handle("POST", "/api/2.2/jobs/runs/delete", func(req testserver.Request) any {
		deletion.settledAtDelete.Store(deletion.settled.Load())
		return testserver.Response{}
	})
	return jobRunClientFor(t, server), &deletion
}

func deleteTestRun(t *testing.T, client *databricks.WorkspaceClient) error {
	t.Helper()
	return (&ResourceJobRun{}).New(client).DoDelete(t.Context(), "123", &JobRunState{})
}

func TestJobRunDeleteCancelsUnfinishedRun(t *testing.T) {
	// An interrupted wait leaves the run going, and jobs/runs/delete rejects it.
	client, deletion := jobRunDeleteClient(t, &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning})

	require.NoError(t, deleteTestRun(t, client))

	assert.True(t, deletion.cancelled.Load(), "expected the run to be cancelled")
	assert.True(t, deletion.settledAtDelete.Load(), "expected the delete to wait for the cancellation to settle")
}

func TestJobRunDeleteLeavesFinishedRunAlone(t *testing.T) {
	client, deletion := jobRunDeleteClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	})

	require.NoError(t, deleteTestRun(t, client))

	assert.False(t, deletion.cancelled.Load(), "a run that already finished has nothing to cancel")
}
