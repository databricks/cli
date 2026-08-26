package dresources

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
)

const jobRunOnValueChangePath = "lifecycle.triggers.on_value_change"

// jobRunTimeout matches the timeout `bundle run` allows a run (bundle/run/job.go).
const jobRunTimeout = 24 * time.Hour

const (
	jobRunValueHashPrefix = "sha256:"
	jobRunValueHashLength = len(jobRunValueHashPrefix) + sha256.Size*2
)

// JobRunTriggersState is the persisted fingerprint of lifecycle.triggers.
type JobRunTriggersState struct {
	// Fresh UUID each plan while armed so Old!=New forces recreate.
	OnBundleDeploy string `json:"on_bundle_deploy,omitempty"`
	// Content hashes from ResolveJobRunFileTriggers; any change recreates.
	OnFileChange map[string]string `json:"on_file_change,omitempty"`
	// Expression → fingerprint. Keyed by the config expression so removing a
	// watch is a dropped key (skip) and a changed value under the same
	// expression recreates. Two watches that resolve to the same value stay distinct.
	OnValueChange map[string]string `json:"on_value_change,omitempty"`
}

// JobRunLifecycleState is the local-only trigger fingerprint. Nested by value,
// not by pointer: structdiff cannot descend into a nil pointer and would report
// the whole subtree at "lifecycle" instead of the leaf that actually changed.
type JobRunLifecycleState struct {
	Triggers JobRunTriggersState `json:"triggers"`
}

func emptyJobRunLifecycleState() JobRunLifecycleState {
	var empty JobRunLifecycleState
	return empty
}

// JobRunState is the RunNow request plus the outcome required for planning.
type JobRunState struct {
	jobs.RunNow

	// Always SUCCESS during planning and cleared before persistence.
	ResultState jobs.RunResultState `json:"result_state,omitempty"`

	// Local-only. Nested under lifecycle to mirror config and avoid colliding
	// with a future Jobs API field.
	Lifecycle JobRunLifecycleState `json:"lifecycle"`
}

func (s *JobRunState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRunState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// JobRunRemote is the RunNow request plus the run's output-only fields. It has no
// lifecycle: GetRun never returns the fingerprints (see knownMissingInRemoteType).
type JobRunRemote struct {
	jobs.RunNow

	// Repeats state.result_state at the path JobRunState uses, so the planner can
	// compare the two. See "RemapState is a dumb copy" in README.md.
	ResultState jobs.RunResultState `json:"result_state,omitempty"`

	RunId      int64          `json:"run_id,omitempty"`
	RunName    string         `json:"run_name,omitempty"`
	State      *jobs.RunState `json:"state,omitempty"`
	RunPageUrl string         `json:"run_page_url,omitempty"`
	RunType    jobs.RunType   `json:"run_type,omitempty"`
}

// Custom marshaler needed because embedded RunNow's MarshalJSON would otherwise
// take over and drop the additional fields.
func (s *JobRunRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRunRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceJobRun struct {
	client *databricks.WorkspaceClient
}

func (*ResourceJobRun) New(client *databricks.WorkspaceClient) *ResourceJobRun {
	return &ResourceJobRun{
		client: client,
	}
}

func (*ResourceJobRun) PrepareState(input *resources.JobRun) *JobRunState {
	state := &JobRunState{
		RunNow:      input.RunNow,
		ResultState: jobs.RunResultStateSuccess,
		Lifecycle:   emptyJobRunLifecycleState(),
	}
	if input.HasOnBundleDeploy() {
		state.Lifecycle.Triggers.OnBundleDeploy = uuid.NewString()
	}
	if len(input.ResolvedFileTriggers) > 0 {
		state.Lifecycle.Triggers.OnFileChange = input.ResolvedFileTriggers
	}
	if len(input.ResolvedValueTriggers) > 0 {
		// Cloned because NormalizeAfterResolve rewrites entries in place.
		state.Lifecycle.Triggers.OnValueChange = maps.Clone(input.ResolvedValueTriggers)
	}
	state.NormalizeAfterResolve()
	return state
}

// PrepareInputConfig puts each still-unresolved watch on its own state path, so
// the deploy graph depends on the watched value rather than the config wrapper.
func (*ResourceJobRun) PrepareInputConfig(input *resources.JobRun, _ string) (*structvar.StructVar, error) {
	refs := map[string]string{}
	parent := structpath.MustParsePath(jobRunOnValueChangePath)
	for expr, value := range input.ResolvedValueTriggers {
		if !dynvar.ContainsVariableReference(value) {
			continue
		}
		refs[structpath.NewBracketString(parent, expr).String()] = value
	}
	return &structvar.StructVar{Value: input, Refs: refs}, nil
}

// NormalizeAfterResolve hashes a watch once it no longer contains a reference.
func (s *JobRunState) NormalizeAfterResolve() {
	for expr, value := range s.Lifecycle.Triggers.OnValueChange {
		if dynvar.ContainsVariableReference(value) {
			continue
		}
		s.Lifecycle.Triggers.OnValueChange[expr] = compactJobRunValue(value)
	}
}

// compactJobRunValue hashes a value only when the digest is shorter than it.
func compactJobRunValue(value string) string {
	if len(value) <= jobRunValueHashLength {
		return value
	}
	sum := sha256.Sum256([]byte(value))
	return jobRunValueHashPrefix + hex.EncodeToString(sum[:])
}

// makeJobRunRemote maps the GetRun response into the RunNow-shaped remote: GET
// nests the params under overriding_parameters and returns job_parameters as a
// list, so both are flattened back into RunNow.
func makeJobRunRemote(run *jobs.Run) *JobRunRemote {
	var overriding jobs.RunParameters
	if run.OverridingParameters != nil {
		overriding = *run.OverridingParameters
	}
	var jobParameters map[string]string
	if len(run.JobParameters) > 0 {
		jobParameters = make(map[string]string, len(run.JobParameters))
		for _, p := range run.JobParameters {
			jobParameters[p.Name] = p.Value
		}
	}
	return &JobRunRemote{
		RunNow: jobs.RunNow{
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
			// Request-only fields GetRun never reports; listed so exhaustruct
			// flags any new SDK field.
			IdempotencyToken:  "",
			Only:              nil,
			PerformanceTarget: "",
			Queue:             nil,
			ForceSendFields:   nil,
		},
		ResultState: run.State.ResultState,
		RunId:       run.RunId,
		RunName:     run.RunName,
		// Rebuilt, not copied: the SDK records explicitly-sent fields in
		// ForceSendFields, which makes zero values workspace-dependent.
		State: &jobs.RunState{
			LifeCycleState: run.State.LifeCycleState,
			ResultState:    run.State.ResultState,
			StateMessage:   run.State.StateMessage,
			// Nothing reads these; listed so exhaustruct flags any new SDK field.
			QueueReason:             "",
			UserCancelledOrTimedout: false,
			ForceSendFields:         nil,
		},
		RunPageUrl: workspaceurls.ModernizeJobRunPageURL(run.RunPageUrl),
		RunType:    run.RunType,
	}
}

// DoRead returns the run as GetRun reports it; a 404 lets the planner re-trigger.
func (r *ResourceJobRun) DoRead(ctx context.Context, id string) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}
	// var + field set (not a literal) avoids listing GetRunRequest's other fields.
	var req jobs.GetRunRequest
	req.RunId = runID
	run, err := r.client.Jobs.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}
	return makeJobRunRemote(run), nil
}

// RemapState extracts the fields used for diffing: the RunNow request and the
// outcome the run reached. Lifecycle has no remote counterpart, so it stays empty
// and the planner skips it as missing_in_remote.
func (*ResourceJobRun) RemapState(remote *JobRunRemote) *JobRunState {
	return &JobRunState{
		RunNow:      remote.RunNow,
		ResultState: remote.ResultState,
		Lifecycle:   emptyJobRunLifecycleState(),
	}
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunRemote, error) {
	// Mint a token so an SDK retry of a lost response returns the same run. Set it
	// on a copy: recording it in state would drift from the empty config and recreate.
	req := config.RunNow
	req.IdempotencyToken = uuid.NewString()
	// RunNow returns only the new run id, so we return a nil remote and let the
	// framework read it back via DoRead.
	wait, err := r.client.Jobs.RunNow(ctx, req)
	if err != nil {
		return "", nil, err
	}
	config.ResultState = ""
	return strconv.FormatInt(wait.RunId, 10), nil, nil
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
	run, err := retries.Poll(ctx, jobRunTimeout, func() (*jobs.Run, *retries.Err) {
		var req jobs.GetRunRequest
		req.RunId = runID
		run, err := r.client.Jobs.GetRun(ctx, req)
		if err != nil {
			return nil, retries.Halt(err)
		}
		if pageURL == "" && run.RunPageUrl != "" {
			pageURL = run.RunPageUrl
			reportRunLine(ctx, runID, "Run URL: "+workspaceurls.ModernizeJobRunPageURL(pageURL))
		}
		if run.State.LifeCycleState != logged.LifeCycleState || run.State.ResultState != logged.ResultState {
			logged = *run.State
			log.Info(ctx, strings.TrimSpace(fmt.Sprintf("job run %d: %s %s", runID, logged.LifeCycleState, logged.ResultState)))
		}
		if !runIsTerminal(run.State.LifeCycleState) {
			return nil, retries.Continues(run.State.StateMessage)
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
	if run.State.ResultState != jobs.RunResultStateSuccess {
		// The error names the outcome.
		return nil, r.runFailedError(ctx, run)
	}
	reportRunLine(ctx, runID, string(run.State.ResultState))
	return makeJobRunRemote(run), nil
}

// runFailedError reports why the run did not succeed, naming each failed task
// and the error it reported.
func (r *ResourceJobRun) runFailedError(ctx context.Context, run *jobs.Run) error {
	// A skipped run has no result_state; report the lifecycle state.
	outcome := cmp.Or(string(run.State.ResultState), string(run.State.LifeCycleState))
	var msg strings.Builder
	// The framework already prefixes the resource key and the run id.
	fmt.Fprintf(&msg, "run did not succeed: %s", outcome)
	if run.State.StateMessage != "" {
		fmt.Fprintf(&msg, ": %s", run.State.StateMessage)
	}
	for _, task := range lastFailedAttempts(run.Tasks) {
		fmt.Fprintf(&msg, "\ntask %q: %s", task.TaskKey, r.taskError(ctx, task))
	}
	msg.WriteString(runPageLine(run.RunPageUrl))
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
		i := slices.IndexFunc(result, func(seen jobs.RunTask) bool { return seen.TaskKey == task.TaskKey })
		switch {
		case i < 0:
			result = append(result, task)
		case task.AttemptNumber > result[i].AttemptNumber:
			result[i] = task
		}
	}
	return result
}

// taskFailed reports whether a task is a cause of the run's failure. A task left
// SKIPPED or UPSTREAM_FAILED never ran, so it has no error to report.
func taskFailed(task jobs.RunTask) bool {
	if task.State != nil {
		return task.State.LifeCycleState == jobs.RunLifeCycleStateInternalError ||
			task.State.ResultState == jobs.RunResultStateFailed ||
			task.State.ResultState == jobs.RunResultStateTimedout
	}
	// State is deprecated in favour of Status, so a workspace may stop setting it.
	if task.Status == nil || task.Status.TerminationDetails == nil {
		return false
	}
	details := task.Status.TerminationDetails
	// SKIPPED is also the code for a task an upstream failure skipped.
	return details.Type != jobs.TerminationTypeTypeSuccess &&
		details.Code != jobs.TerminationCodeCodeSkipped
}

// taskError returns the message the task reported, from the same place `bundle
// run` reads it.
func (r *ResourceJobRun) taskError(ctx context.Context, task jobs.RunTask) string {
	var reported string
	output, err := r.client.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{RunId: task.RunId})
	if err != nil {
		log.Debugf(ctx, "could not read output of task %s: %v", task.TaskKey, err)
	} else {
		reported = output.Error
	}
	// Not every task type reports an error through GetRunOutput.
	return cmp.Or(reported, taskMessage(task))
}

// taskMessage reads the message off whichever field taskFailed accepted the task
// on, so the field it reads is always set.
func taskMessage(task jobs.RunTask) string {
	if task.State != nil {
		return task.State.StateMessage
	}
	return task.Status.TerminationDetails.Message
}

func runPageLine(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	return "\nrun page: " + workspaceurls.ModernizeJobRunPageURL(rawURL)
}

// runIsTerminal reports whether a run has stopped, whatever it stopped as.
func runIsTerminal(state jobs.RunLifeCycleState) bool {
	return state == jobs.RunLifeCycleStateTerminated ||
		state == jobs.RunLifeCycleStateSkipped ||
		state == jobs.RunLifeCycleStateInternalError
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
// Clearing a trigger skips its local-only fingerprint without re-firing the run.
func (*ResourceJobRun) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, remote *JobRunRemote) error {
	switch path.String() {
	case "lifecycle.triggers.on_bundle_deploy":
		if change.New == nil || change.New == "" {
			change.Action = deployplan.Skip
			change.Reason = "trigger removed"
		}
	case "lifecycle.triggers.on_file_change":
		// Only a cleared trigger skips: a file dropping out of the map means the
		// match disappeared, which is a real change.
		if change.New == nil {
			change.Action = deployplan.Skip
			change.Reason = "trigger removed"
		}
	case jobRunOnValueChangePath:
		if valueTriggersOnlyRemoved(change.Old, change.New) {
			change.Action = deployplan.Skip
			change.Reason = "trigger removed"
		}
	case "result_state":
		// The planner passes no remote state when the run could not be read.
		if remote == nil || runIsTerminal(remote.State.LifeCycleState) {
			return nil
		}
		change.Action = deployplan.Skip
		change.Reason = "run in progress"
	default:
		// structdiff reports the same removal again under each dropped key. A
		// changed fingerprint under a key that stayed keeps its recreate.
		parent := path.Parent()
		if parent != nil && parent.String() == jobRunOnValueChangePath && change.New == nil {
			change.Action = deployplan.Skip
			change.Reason = "trigger removed"
		}
	}
	return nil
}

// valueTriggersOnlyRemoved reports whether new is old minus one or more watches,
// with every remaining fingerprint unchanged. Keying state by the config
// expression is what makes this exact: dropping a watch is a dropped key, and
// changing what a watch resolves to is a new value under the same key, even when
// that value is the one a just-removed watch used to hold.
func valueTriggersOnlyRemoved(oldValue, newValue any) bool {
	if newValue == nil {
		return true
	}
	oldMap, okOld := oldValue.(map[string]string)
	newMap, okNew := newValue.(map[string]string)
	if !okOld || !okNew {
		return false
	}
	for expr, value := range newMap {
		if old, ok := oldMap[expr]; !ok || old != value {
			return false
		}
	}
	return len(newMap) < len(oldMap)
}

// DoDelete deletes the run via jobs/runs/delete, on both destroy and the
// recreate path. The API rejects a still-active run, which an interrupted wait
// leaves behind, so cancel it first.
func (r *ResourceJobRun) DoDelete(ctx context.Context, id string, _ *JobRunState) error {
	runID, err := parseRunID(id)
	if err != nil {
		return err
	}
	remote, err := r.DoRead(ctx, id)
	if err != nil {
		return err
	}
	if !runIsTerminal(remote.State.LifeCycleState) {
		err = r.cancelRun(ctx, runID)
		if err != nil {
			return err
		}
	}
	return r.client.Jobs.DeleteRunByRunId(ctx, runID)
}

// cancelRun cancels a run and waits for it to settle, since cancellation is
// asynchronous and the delete that follows needs a settled run.
func (r *ResourceJobRun) cancelRun(ctx context.Context, runID int64) error {
	waiter, err := r.client.Jobs.CancelRun(ctx, jobs.CancelRun{RunId: runID})
	if err != nil {
		return fmt.Errorf("cancelling run %d before deleting it: %w", runID, err)
	}
	_, err = waiter.Get()
	if err != nil {
		return fmt.Errorf("waiting for run %d to be cancelled: %w", runID, err)
	}
	return nil
}

func parseRunID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: run id is not integer: %q: %w", id, err)
	}
	return result, nil
}
