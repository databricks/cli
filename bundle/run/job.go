package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/poll"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/run/progress"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/cli/libs/workspaceurls"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// Default timeout for waiting for a job run to complete.
var jobRunTimeout time.Duration = 24 * time.Hour

type jobRunner struct {
	key

	bundle *bundle.Bundle
	job    *resources.Job
}

func (r *jobRunner) Name() string {
	if r.job == nil {
		return ""
	}
	return utils.Deref(r.job.Name)
}

func taskState(task sdkjobs.RunTask) sdkjobs.RunState {
	if task.State == nil {
		return sdkjobs.RunState{}
	}
	return *task.State
}

func isFailed(task sdkjobs.RunTask) bool {
	state := taskState(task)
	return state.LifeCycleState == jobs.RunLifeCycleState_RunLifeCycleState_InternalError ||
		(state.LifeCycleState == jobs.RunLifeCycleState_RunLifeCycleState_Terminated &&
			state.ResultState == jobs.RunResultState_RunResultState_Failed)
}

func isSuccess(task sdkjobs.RunTask) bool {
	state := taskState(task)
	return state.LifeCycleState == jobs.RunLifeCycleState_RunLifeCycleState_Terminated &&
		state.ResultState == jobs.RunResultState_RunResultState_Success
}

func (r *jobRunner) logFailedTasks(ctx context.Context, runId int64) {
	w := r.bundle.WorkspaceClient(ctx)
	run, err := output.GetRun(ctx, w, runId)
	if err != nil {
		log.Errorf(ctx, "failed to log job run. Error: %s", err)
		return
	}
	if run.State != nil && run.State.ResultState == jobs.RunResultState_RunResultState_Success {
		return
	}
	for _, task := range run.Tasks {
		taskKey := utils.Deref(task.TaskKey)
		if isSuccess(task) {
			log.Infof(ctx, "task %s completed successfully", cmdio.Green(ctx, taskKey))
		} else if isFailed(task) {
			taskInfo, err := w.Jobs.GetRunOutput(ctx, sdkjobs.GetRunOutputRequest{
				RunId: task.RunId,
			})
			if err != nil {
				log.Errorf(ctx, "task %s failed. Unable to fetch error trace: %s", cmdio.Red(ctx, taskKey), err)
				continue
			}
			taskError := utils.Deref(taskInfo.Error)
			taskTrace := utils.Deref(taskInfo.ErrorTrace)
			cmdio.Log(ctx, progress.NewTaskErrorEvent(taskKey, taskError, taskTrace))
			log.Errorf(ctx, "Task %s failed!\nError:\n%s\nTrace:\n%s",
				cmdio.Red(ctx, taskKey), taskError, taskTrace)
		} else {
			log.Infof(ctx, "task %s is in state %s",
				cmdio.Yellow(ctx, taskKey), taskState(task).LifeCycleState)
		}
	}
}

// jobRunMonitor tracks state for a single job run and provides callbacks
// for monitoring progress.
type jobRunMonitor struct {
	ctx       context.Context
	prevState *sdkjobs.RunState
}

// onProgress is the single callback that handles all state tracking and logging.
func (m *jobRunMonitor) onProgress(info *sdkjobs.GetRunResponse) {
	state := info.State
	if state == nil {
		return
	}

	// First time we see this run.
	if m.prevState == nil {
		runURL := workspaceurls.ModernizeJobRunPageURL(utils.Deref(info.RunPageUrl))
		log.Infof(m.ctx, "Run available at %s", runURL)
		cmdio.Log(m.ctx, progress.NewJobRunUrlEvent(runURL))
	}

	// No state change: do not log.
	if m.prevState != nil &&
		m.prevState.LifeCycleState == state.LifeCycleState &&
		m.prevState.ResultState == state.ResultState {
		return
	}

	// Capture current state as previous state for next call.
	m.prevState = state

	runState, err := jobs.RunStateFromSDK(state)
	if err != nil {
		log.Warnf(m.ctx, "cannot convert run state: %s", err)
		return
	}

	// Log progress event both to the terminal (in place or append), and to the logger.
	event := &progress.JobProgressEvent{
		Timestamp: time.Now(),
		JobId:     utils.Deref(info.JobId),
		RunId:     utils.Deref(info.RunId),
		RunName:   utils.Deref(info.RunName),
		State:     *runState,
	}
	cmdio.Log(m.ctx, event)
	log.Info(m.ctx, event.String())
}

func (r *jobRunner) Run(ctx context.Context, opts *Options) (output.RunOutput, error) {
	jobID, err := strconv.ParseInt(r.job.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("job ID is not an integer: %s", r.job.ID)
	}

	err = r.convertPythonParams(opts)
	if err != nil {
		return nil, err
	}

	// construct request payload from cmd line flags args
	req, err := opts.Job.toPayload(r.job, jobID)
	if err != nil {
		return nil, err
	}

	// Include resource key in logger.
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("resource", r.Key()))

	w := r.bundle.WorkspaceClient(ctx)

	monitor := &jobRunMonitor{
		ctx: ctx,
	}

	waiter, err := w.Jobs.RunNow(ctx, *req)
	if err != nil {
		return nil, fmt.Errorf("cannot start job: %w", err)
	}
	runId := waiter.GetRunId()

	if opts.NoWait {
		details, err := w.Jobs.GetRun(ctx, sdkjobs.GetRunRequest{
			RunId: new(runId),
		})
		if err != nil {
			return nil, err
		}
		cmdio.Log(ctx, progress.NewJobRunUrlEvent(workspaceurls.ModernizeJobRunPageURL(utils.Deref(details.RunPageUrl))))
		return nil, nil
	}

	run, err := waitForRun(ctx, w, runId, jobRunTimeout, monitor.onProgress)
	if err != nil {
		r.logFailedTasks(ctx, runId)
	}
	if err != nil {
		return nil, err
	}
	stateMessage := utils.Deref(run.State.StateMessage)
	if run.State.LifeCycleState == jobs.RunLifeCycleState_RunLifeCycleState_Skipped {
		log.Infof(ctx, "Run was skipped!")
		return nil, fmt.Errorf("run skipped: %s", stateMessage)
	}

	switch run.State.ResultState {
	// The run was canceled at user request.
	case jobs.RunResultState_RunResultState_Canceled:
		log.Infof(ctx, "Run was cancelled!")
		return nil, fmt.Errorf("run canceled: %s", stateMessage)

	// The task completed with an error.
	case jobs.RunResultState_RunResultState_Failed:
		log.Infof(ctx, "Run has failed!")
		return nil, fmt.Errorf("run failed: %s", stateMessage)

	// The task completed successfully.
	case jobs.RunResultState_RunResultState_Success:
		log.Infof(ctx, "Run has completed successfully!")
		return output.GetJobOutput(ctx, r.bundle.WorkspaceClient(ctx), runId)

	// The run was stopped after reaching the timeout.
	case jobs.RunResultState_RunResultState_Timedout:
		log.Infof(ctx, "Run has timed out!")
		return nil, fmt.Errorf("run timed out: %s", stateMessage)

	// TODO: handle other result states.
	default:
	}

	return nil, err
}

func (r *jobRunner) convertPythonParams(opts *Options) error {
	if r.bundle.Config.Experimental != nil && !r.bundle.Config.Experimental.PythonWheelWrapper {
		return nil
	}

	needConvert := false
	for _, task := range r.job.Tasks {
		if task.PythonWheelTask != nil {
			needConvert = true
			break
		}
	}

	if !needConvert {
		return nil
	}

	if len(opts.Job.pythonParams) == 0 {
		return nil
	}

	if opts.Job.notebookParams == nil {
		opts.Job.notebookParams = make(map[string]string)
	}

	if len(opts.Job.pythonParams) > 0 {
		if _, ok := opts.Job.notebookParams["__python_params"]; ok {
			return errors.New("can't use __python_params as notebook param, the name is reserved for internal use")
		}
		p, err := json.Marshal(opts.Job.pythonParams)
		if err != nil {
			return err
		}
		opts.Job.notebookParams["__python_params"] = string(p)
	}

	return nil
}

func (r *jobRunner) Cancel(ctx context.Context) error {
	w := r.bundle.WorkspaceClient(ctx)
	jobID, err := strconv.ParseInt(r.job.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("job ID is not an integer: %s", r.job.ID)
	}

	var runs []*sdkjobs.BaseRun
	for run, err := range w.Jobs.ListRunsIter(ctx, sdkjobs.ListRunsRequest{
		StateConstraint: &sdkjobs.ListRunsRequest_StateConstraint_ActiveOnly{ActiveOnly: true},
		JobId:           new(jobID),
	}) {
		if err != nil {
			return err
		}
		runs = append(runs, run)
	}

	if len(runs) == 0 {
		return nil
	}

	errGroup, errCtx := errgroup.WithContext(ctx)
	for _, run := range runs {
		runId := run.RunId
		errGroup.Go(func() error {
			wait, err := w.Jobs.CancelRun(errCtx, sdkjobs.CancelRunRequest{
				RunId: runId,
			})
			if err != nil {
				return err
			}
			// Waits for the Terminated or Skipped state
			_, err = wait.Wait(errCtx, lro.WithTimeout(jobRunTimeout))
			return err
		})
	}

	return errGroup.Wait()
}

func (r *jobRunner) Restart(ctx context.Context, opts *Options) (output.RunOutput, error) {
	// We don't need to cancel existing runs if the job is continuous and unpaused.
	// the /jobs/run-now API will automatically cancel any existing runs before starting a new one.
	//
	// /jobs/run-now will not cancel existing runs if the job is continuous and paused.
	// New job runs will be queued instead and will wait for existing runs to finish.
	// In this case, we need to cancel the existing runs before starting a new one.
	continuous := r.job.Continuous
	if continuous != nil && continuous.PauseStatus == pauseStatusUnpaused {
		return r.Run(ctx, opts)
	}

	sp := cmdio.NewSpinner(ctx)
	sp.Update("Cancelling all active job runs")
	err := r.Cancel(ctx)
	sp.Close()
	if err != nil {
		return nil, err
	}

	return r.Run(ctx, opts)
}

func (r *jobRunner) ParseArgs(args []string, opts *Options) error {
	return r.posArgsHandler().ParseArgs(args, opts)
}

func (r *jobRunner) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return r.posArgsHandler().CompleteArgs(args, toComplete)
}

// sdk-go only names PAUSED (UNPAUSED is the proto default), but configurations
// and the API use the explicit "UNPAUSED" value.
const pauseStatusUnpaused jobs.SchedulePauseStatus = "UNPAUSED"

// waitForRun polls the run until it is TERMINATED or SKIPPED, calling
// onProgress after every poll, with the error messages of the previous SDK's
// run waiter.
func waitForRun(ctx context.Context, w *sdkclients.Clients, runId int64, timeout time.Duration, onProgress func(*sdkjobs.GetRunResponse)) (*sdkjobs.GetRunResponse, error) {
	return poll.Until(ctx, timeout, func(ctx context.Context) (poll.Status[sdkjobs.GetRunResponse], error) {
		run, err := output.GetRun(ctx, w, runId)
		if err != nil {
			return poll.Status[sdkjobs.GetRunResponse]{}, err
		}
		onProgress(run)
		var state sdkjobs.RunState
		if run.State != nil {
			state = *run.State
		}
		statusMessage := utils.Deref(state.StateMessage)
		switch state.LifeCycleState {
		case jobs.RunLifeCycleState_RunLifeCycleState_Terminated, jobs.RunLifeCycleState_RunLifeCycleState_Skipped:
			return poll.Status[sdkjobs.GetRunResponse]{Done: true, Result: run}, nil
		case jobs.RunLifeCycleState_RunLifeCycleState_InternalError:
			return poll.Status[sdkjobs.GetRunResponse]{}, fmt.Errorf("failed to reach %s or %s, got %s: %s",
				jobs.RunLifeCycleState_RunLifeCycleState_Terminated, jobs.RunLifeCycleState_RunLifeCycleState_Skipped,
				state.LifeCycleState, statusMessage)
		case jobs.RunLifeCycleState_RunLifeCycleState_Unspecified, jobs.RunLifeCycleState_RunLifeCycleState_Pending,
			jobs.RunLifeCycleState_RunLifeCycleState_Running, jobs.RunLifeCycleState_RunLifeCycleState_Terminating,
			jobs.RunLifeCycleState_RunLifeCycleState_Blocked, jobs.RunLifeCycleState_RunLifeCycleState_WaitingForRetry,
			jobs.RunLifeCycleState_RunLifeCycleState_Queued:
			// Still running or queued; keep polling
		}
		return poll.Status[sdkjobs.GetRunResponse]{Message: statusMessage}, nil
	})
}
