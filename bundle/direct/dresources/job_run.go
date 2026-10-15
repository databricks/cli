package dresources

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	jobs "github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/cli/libs/workspaceurls"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
	"github.com/google/uuid"
)

// jobRunTimeout matches the timeout `bundle run` allows a run (bundle/run/job.go).
const jobRunTimeout = 24 * time.Hour

// JobRunLifecycleState is the local-only trigger fingerprint.
type JobRunLifecycleState struct {
	TriggersState *resources.JobRunTriggersState `json:"triggers_state,omitempty"`
}

// JobRunState is the RunNow request plus the outcome required for planning.
type JobRunState struct {
	jobs.RunNowRequest

	// Always SUCCESS during planning and cleared before persistence.
	ResultState jobs.RunResultState_RunResultState `json:"result_state,omitempty"`

	// Local-only fingerprint. Nested under lifecycle to avoid colliding with a
	// future Jobs API field; nil when no trigger is armed so the key is omitted.
	Lifecycle *JobRunLifecycleState `json:"lifecycle,omitempty"`
}

// JobRunRemote is the RunNow request plus the run's output-only fields. It has no
// lifecycle: GetRun never returns the fingerprints (see knownMissingInRemoteType).
type JobRunRemote struct {
	jobs.RunNowRequest

	// Repeats state.result_state at the path JobRunState uses, so the planner can
	// compare the two. See "RemapState is a dumb copy" in README.md.
	ResultState jobs.RunResultState_RunResultState `json:"result_state,omitempty"`

	RunId      int64          `json:"run_id,omitempty"`
	RunName    string         `json:"run_name,omitempty"`
	State      *jobs.RunState `json:"state,omitempty"`
	RunPageUrl string         `json:"run_page_url,omitempty"`
	RunType    jobs.RunType   `json:"run_type,omitempty"`
}

type ResourceJobRun struct {
	client *sdkclients.Clients
}

func (*ResourceJobRun) New(client *sdkclients.Clients) *ResourceJobRun {
	return &ResourceJobRun{
		client: client,
	}
}

func (*ResourceJobRun) PrepareState(input *resources.JobRun) *JobRunState {
	state := &JobRunState{
		RunNowRequest: input.RunNowRequest,
		ResultState:   jobs.RunResultState_RunResultState_Success,
		Lifecycle:     nil,
	}
	var ts resources.JobRunTriggersState
	if input.Lifecycle != nil && input.Lifecycle.TriggersState != nil {
		ts = *input.Lifecycle.TriggersState
	}
	if input.HasOnBundleDeploy() {
		ts.OnBundleDeploy = uuid.NewString()
	}
	if !ts.IsEmpty() {
		state.Lifecycle = &JobRunLifecycleState{TriggersState: &ts}
	}
	return state
}

// makeJobRunRemote maps the GetRun response into the RunNow-shaped remote: GET
// nests the params under overriding_parameters and returns job_parameters as a
// list, so both are flattened back into RunNow.
func makeJobRunRemote(run *jobs.GetRunResponse) *JobRunRemote {
	var overriding jobs.RunParameters
	if run.OverridingParameters != nil {
		overriding = *run.OverridingParameters
	}
	var jobParameters map[string]string
	if len(run.JobParameters) > 0 {
		jobParameters = make(map[string]string, len(run.JobParameters))
		for _, p := range run.JobParameters {
			jobParameters[utils.Deref(p.Name)] = utils.Deref(p.Value)
		}
	}
	state := runState(run)
	return &JobRunRemote{
		JobId:             run.JobId,
		JobParameters:     jobParameters,
		DbtCommands:       overriding.DbtCommands,
		JarParams:         overriding.JarParams,
		NotebookParams:    overriding.NotebookParams,
		PipelineParams:    overriding.PipelineParams,
		PythonNamedParams: overriding.PythonNamedParams,
		PythonParams:      overriding.PythonParams,
		SparkSubmitParams: overriding.SparkSubmitParams,
		SqlParams:         overriding.SqlParams,
		// Request-only fields GetRun never reports.
		IdempotencyToken:  nil,
		Only:              nil,
		PerformanceTarget: "",
		Queue:             nil,
		ResultState:       state.ResultState,
		RunId:             utils.Deref(run.RunId),
		RunName:           utils.Deref(run.RunName),
		// Only the fields planning reads; queue_reason and user_cancelled_or_timedout
		// are left out so the remote does not depend on what the workspace reports.
		State: &jobs.RunState{
			LifeCycleState:          state.LifeCycleState,
			ResultState:             state.ResultState,
			StateMessage:            state.StateMessage,
			QueueReason:             nil, // Not set by DABs
			UserCancelledOrTimedout: nil, // Not set by DABs
		},
		RunPageUrl: workspaceurls.ModernizeJobRunPageURL(utils.Deref(run.RunPageUrl)),
		RunType:    run.RunType,
	}
}

// runState returns the run's state, or an empty one if the run has none.
func runState(run *jobs.GetRunResponse) jobs.RunState {
	if run.State == nil {
		return jobs.RunState{
			LifeCycleState:          "",  // Not set by DABs
			QueueReason:             nil, // Not set by DABs
			ResultState:             "",  // Not set by DABs
			StateMessage:            nil, // Not set by DABs
			UserCancelledOrTimedout: nil, // Not set by DABs
		}
	}
	return *run.State
}

func (r *ResourceJobRun) getRun(ctx context.Context, runID int64) (*jobs.GetRunResponse, error) {
	run, err := r.client.Jobs.GetRun(ctx, sdkjobs.GetRunRequest{
		RunId:                 new(runID),
		IncludeHistory:        nil, // Not set by DABs
		IncludeResolvedValues: nil, // Not set by DABs
		PageToken:             nil, // Not set by DABs
	})
	if err != nil {
		return nil, err
	}
	return jobs.GetRunResponseFromSDK(run)
}

// DoRead returns the run as GetRun reports it; a 404 lets the planner re-trigger.
func (r *ResourceJobRun) DoRead(ctx context.Context, id string) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}
	run, err := r.getRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	return makeJobRunRemote(run), nil
}

// RemapState extracts the fields used for diffing: the RunNow request and the
// outcome the run reached. Lifecycle has no remote counterpart.
func (*ResourceJobRun) RemapState(remote *JobRunRemote) *JobRunState {
	return &JobRunState{
		RunNowRequest: remote.RunNowRequest,
		ResultState:   remote.ResultState,
		Lifecycle:     nil,
	}
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunRemote, error) {
	// Mint a token so an SDK retry of a lost response returns the same run. Set it
	// on a copy: recording it in state would drift from the empty config and recreate.
	req := config.RunNowRequest
	req.IdempotencyToken = new(uuid.NewString())
	request, err := jobs.RunNowRequestToSDK(&req)
	if err != nil {
		return "", nil, err
	}
	// RunNow returns only the new run id, so we return a nil remote and let the
	// framework read it back via DoRead.
	wait, err := r.client.Jobs.RunNow(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	config.ResultState = ""
	return strconv.FormatInt(wait.GetRunId(), 10), nil, nil
}

// WaitAfterCreate blocks until the run finishes, so a resource referencing its
// output (e.g. state.result_state) sees a settled run. Only SUCCESS continues the deploy.
func (r *ResourceJobRun) WaitAfterCreate(ctx context.Context, id string, _ *JobRunState) (*JobRunRemote, error) {
	return r.waitForRun(ctx, id)
}

// waitForRun polls the run until it stops, and fails unless it succeeded.
func (r *ResourceJobRun) waitForRun(ctx context.Context, id string) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}

	// A run can take hours, so report the run page as soon as it is known. pageURL
	// outlives the poll so an abandoned wait can still link the run.
	var pageURL string
	// Every state is logged, but only the outcome is reported: how many states a
	// run passes through varies with its compute, and output has to stay reproducible.
	var logged jobs.RunState
	// Not the SDK's WaitGetRunJobTerminatedOrSkipped: it discards the run on
	// INTERNAL_ERROR, the state a run whose task failed lands in, leaving
	// runFailedError nothing to name the failing task from. This poll returns the
	// run in whatever state it settles in.
	run, err := retries.Poll(ctx, jobRunTimeout, func() (*jobs.GetRunResponse, *retries.Err) {
		run, err := r.getRun(ctx, runID)
		if err != nil {
			return nil, retries.Halt(err)
		}
		if pageURL == "" && utils.Deref(run.RunPageUrl) != "" {
			pageURL = *run.RunPageUrl
			reportRunLine(ctx, runID, "Run URL: "+workspaceurls.ModernizeJobRunPageURL(pageURL))
		}
		state := runState(run)
		if state.LifeCycleState != logged.LifeCycleState || state.ResultState != logged.ResultState {
			logged = state
			log.Info(ctx, strings.TrimSpace(fmt.Sprintf("job run %d: %s %s", runID, logged.LifeCycleState, logged.ResultState)))
		}
		if !runIsTerminal(state.LifeCycleState) {
			return nil, retries.Continues(utils.Deref(state.StateMessage))
		}
		return run, nil
	})
	if err != nil {
		// The wait can end with the run still going (timeout, interrupt), so link
		// the run page.
		if ctx.Err() != nil {
			// A cancelled context is reported as a timeout.
			return nil, fmt.Errorf("interrupted while waiting for the run to finish: %w%s", ctx.Err(), runPageLine(pageURL))
		}
		return nil, fmt.Errorf("%w%s", err, runPageLine(pageURL))
	}
	if runState(run).ResultState != jobs.RunResultState_RunResultState_Success {
		// The error names the outcome.
		return nil, r.runFailedError(ctx, run)
	}
	reportRunLine(ctx, runID, string(runState(run).ResultState))
	return makeJobRunRemote(run), nil
}

// runFailedError reports why the run did not succeed, naming each failed task
// and the error it reported.
func (r *ResourceJobRun) runFailedError(ctx context.Context, run *jobs.GetRunResponse) error {
	state := runState(run)
	// A skipped run has no result_state; report the lifecycle state.
	outcome := cmp.Or(string(state.ResultState), string(state.LifeCycleState))
	var msg strings.Builder
	// The framework already prefixes the resource key and the run id.
	fmt.Fprintf(&msg, "run did not succeed: %s", outcome)
	if message := utils.Deref(state.StateMessage); message != "" {
		fmt.Fprintf(&msg, ": %s", message)
	}
	for _, task := range lastFailedAttempts(run.Tasks) {
		fmt.Fprintf(&msg, "\ntask %q: %s", utils.Deref(task.TaskKey), r.taskError(ctx, task))
	}
	msg.WriteString(runPageLine(utils.Deref(run.RunPageUrl)))
	return errors.New(msg.String())
}

// lastFailedAttempts returns the failed tasks in the order the run reports them,
// one per task key: a task the Jobs API retried is reported once per attempt, and
// only its last one says how the run ended up.
func lastFailedAttempts(tasks []jobs.RunTask) []jobs.RunTask {
	var result []jobs.RunTask
	for _, task := range tasks {
		if !taskFailed(task) {
			continue
		}
		i := slices.IndexFunc(result, func(seen jobs.RunTask) bool { return utils.Deref(seen.TaskKey) == utils.Deref(task.TaskKey) })
		switch {
		case i < 0:
			result = append(result, task)
		case utils.Deref(task.AttemptNumber) > utils.Deref(result[i].AttemptNumber):
			result[i] = task
		}
	}
	return result
}

// taskFailed reports whether a task is a cause of the run's failure. A task left
// SKIPPED or UPSTREAM_FAILED never ran, so it has no error to report.
func taskFailed(task jobs.RunTask) bool {
	if task.State != nil {
		return task.State.LifeCycleState == jobs.RunLifeCycleState_RunLifeCycleState_InternalError ||
			task.State.ResultState == jobs.RunResultState_RunResultState_Failed ||
			task.State.ResultState == jobs.RunResultState_RunResultState_Timedout
	}
	// State is deprecated in favour of Status, so a workspace may stop setting it.
	if task.Status == nil || task.Status.TerminationDetails == nil {
		return false
	}
	details := task.Status.TerminationDetails
	// SKIPPED is also the code for a task an upstream failure skipped.
	return details.Type != jobs.TerminationType_Type_Success &&
		details.Code != jobs.TerminationCode_Code_Skipped
}

// taskError returns the message the task reported, from the same place `bundle
// run` reads it.
func (r *ResourceJobRun) taskError(ctx context.Context, task jobs.RunTask) string {
	var reported string
	output, err := r.client.Jobs.GetRunOutput(ctx, sdkjobs.GetRunOutputRequest{RunId: task.RunId})
	if err != nil {
		log.Debugf(ctx, "could not read output of task %s: %v", utils.Deref(task.TaskKey), err)
	} else {
		reported = utils.Deref(output.Error)
	}
	// Not every task type reports an error through GetRunOutput.
	return cmp.Or(reported, taskMessage(task))
}

// taskMessage reads the message off whichever field taskFailed accepted the task
// on, so the field it reads is always set.
func taskMessage(task jobs.RunTask) string {
	if task.State != nil {
		return utils.Deref(task.State.StateMessage)
	}
	return utils.Deref(task.Status.TerminationDetails.Message)
}

func runPageLine(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	return "\nrun page: " + workspaceurls.ModernizeJobRunPageURL(rawURL)
}

// runIsTerminal reports whether a run has stopped, whatever it stopped as.
func runIsTerminal(state jobs.RunLifeCycleState_RunLifeCycleState) bool {
	return state == jobs.RunLifeCycleState_RunLifeCycleState_Terminated ||
		state == jobs.RunLifeCycleState_RunLifeCycleState_Skipped ||
		state == jobs.RunLifeCycleState_RunLifeCycleState_InternalError
}

// reportRunLine names the resource and run id so concurrent deploys stay readable.
// Deploy attaches the key via [WithResourceKey].
func reportRunLine(ctx context.Context, runID int64, msg string) {
	if !cmdio.HasIO(ctx) {
		return
	}
	cmdio.LogString(ctx, fmt.Sprintf("Output from %s: id=%d: %s", ResourceKey(ctx), runID, msg))
}

// OverrideChangeDesc downgrades result_state drift to skip while the run is
// still going, so a run that may yet succeed is not recreated. A run that
// stopped without succeeding keeps its recreate. A SKIPPED run reports no
// result_state either, so the lifecycle state is what tells the two apart.
// Removing a trigger drops the change and leaves the last fingerprint in state,
// so re-adding the trigger only re-fires when the watched files changed
// meanwhile. All other trigger changes recreate the run.
func (*ResourceJobRun) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, remote *JobRunRemote) error {
	switch path.String() {
	case "lifecycle":
		// Dropped when the trigger is removed (New nil) and when both sides are
		// present, where the trigger fields below classify the change instead.
		// Arming from no lifecycle at all keeps its recreate.
		if change.New == nil || change.Old != nil {
			change.Reason = deployplan.ReasonDrop
		}
	case "lifecycle.triggers_state.on_bundle_deploy":
		if change.New == nil || change.New == "" {
			change.Reason = deployplan.ReasonDrop
		}
	case "lifecycle.triggers_state.on_file_change":
		// As above: an emptied map is a removal, and pattern entries classify a map
		// that still has both sides.
		if isEmptyFileTriggerMap(change.New) || change.Old != nil {
			change.Reason = deployplan.ReasonDrop
		}
	case "result_state":
		// The planner passes no remote state when the run could not be read.
		if remote == nil || runIsTerminal(remote.State.LifeCycleState) {
			return nil
		}
		change.Action = deployplan.Skip
		change.Reason = "run in progress"
	default:
		// A single on_file_change pattern entry, e.g.
		// lifecycle.triggers_state.on_file_change['seed.txt']. Removing one pattern
		// drops its change and leaves the last fingerprint in state.
		if parent := path.Parent(); parent != nil && parent.String() == "lifecycle.triggers_state.on_file_change" {
			if change.New == nil {
				change.Reason = deployplan.ReasonDrop
			}
		}
	}
	return nil
}

// ResourceJobRun intentionally implements no DoDelete: a run is a historical
// record, so removing it from the bundle (or recreating it when a trigger
// re-fires) leaves the run alone on the backend and only drops the state entry
// (see PlanEntry.StateOnly). A recreate still triggers a fresh run.

func isEmptyFileTriggerMap(v any) bool {
	if v == nil {
		return true
	}
	m, ok := v.(map[string]string)
	return ok && len(m) == 0
}

func parseRunID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: run id is not integer: %q: %w", id, err)
	}
	return result, nil
}
