package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/run/progress"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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
	return r.job.Name
}

func isFailed(task jobs.RunTask) bool {
	return task.State.LifeCycleState == jobs.RunLifeCycleStateInternalError ||
		(task.State.LifeCycleState == jobs.RunLifeCycleStateTerminated &&
			task.State.ResultState == jobs.RunResultStateFailed)
}

func isSuccess(task jobs.RunTask) bool {
	return task.State.LifeCycleState == jobs.RunLifeCycleStateTerminated &&
		task.State.ResultState == jobs.RunResultStateSuccess
}

func (r *jobRunner) logFailedTasks(ctx context.Context, runId int64) {
	w := r.bundle.WorkspaceClient(ctx)
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{
		RunId: runId,
	})
	if err != nil {
		log.Errorf(ctx, "failed to log job run. Error: %s", err)
		return
	}
	if run.State.ResultState == jobs.RunResultStateSuccess {
		return
	}
	for _, task := range run.Tasks {
		if isSuccess(task) {
			log.Infof(ctx, "task %s completed successfully", cmdio.Green(ctx, task.TaskKey))
		} else if isFailed(task) {
			taskInfo, err := w.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{
				RunId: task.RunId,
			})
			if err != nil {
				log.Errorf(ctx, "task %s failed. Unable to fetch error trace: %s", cmdio.Red(ctx, task.TaskKey), err)
				continue
			}
			cmdio.Log(ctx, progress.NewTaskErrorEvent(task.TaskKey, taskInfo.Error, taskInfo.ErrorTrace))
			log.Errorf(ctx, "Task %s failed!\nError:\n%s\nTrace:\n%s",
				cmdio.Red(ctx, task.TaskKey), taskInfo.Error, taskInfo.ErrorTrace)
		} else {
			log.Infof(ctx, "task %s is in state %s",
				cmdio.Yellow(ctx, task.TaskKey), task.State.LifeCycleState)
		}
	}
}

// jobRunMonitor tracks state for a single job run and provides callbacks
// for monitoring progress.
type jobRunMonitor struct {
	ctx       context.Context
	prevState *jobs.RunState
}

// onProgress is the single callback that handles all state tracking and logging.
func (m *jobRunMonitor) onProgress(info *jobs.Run) {
	state := info.State
	if state == nil {
		return
	}

	// First time we see this run.
	if m.prevState == nil {
		runURL := runPageURL(m.ctx, info.RunPageUrl)
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

	// Log progress event both to the terminal (in place or append), and to the logger.
	event := &progress.JobProgressEvent{
		Timestamp: time.Now(),
		JobId:     info.JobId,
		RunId:     info.RunId,
		RunName:   info.RunName,
		State:     *info.State,
	}
	cmdio.Log(m.ctx, event)
	log.Info(m.ctx, event.String())
}

// runPageURL converts the legacy run URL returned by the Jobs API
//
//	https://<host>/?o=<id>#job/<jobID>/run/<runID>
//
// into the modern path form
//
//	https://<host>/jobs/<jobID>/runs/<runID>?o=<id>
//
// so that non-admin users permitted to view the run are not redirected to the
// workspace homepage. See https://github.com/databricks/cli/issues/5142. The
// workspace selector query param (o) is preserved as-is. The conversion is
// cosmetic, so the original URL is returned on the rare chance the format is
// unexpected.
func runPageURL(ctx context.Context, raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		log.Debugf(ctx, "could not parse run URL %q: %v", raw, err)
		return raw
	}

	jobID, runID, ok := parseLegacyRunFragment(u.Fragment)
	if !ok {
		log.Debugf(ctx, "unexpected run URL fragment %q", u.Fragment)
		return raw
	}

	u.Fragment = ""
	u.Path = "/" + workspaceurls.JobRunPath(jobID, runID)
	return u.String()
}

// parseLegacyRunFragment extracts the job and run IDs from a legacy run URL
// fragment of the form "job/<jobID>/run/<runID>".
func parseLegacyRunFragment(fragment string) (jobID, runID string, ok bool) {
	parts := strings.Split(fragment, "/")
	if len(parts) != 4 || parts[0] != "job" || parts[2] != "run" || parts[1] == "" || parts[3] == "" {
		return "", "", false
	}
	return parts[1], parts[3], true
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

	if opts.NoWait {
		details, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{
			RunId: waiter.RunId,
		})
		if err != nil {
			return nil, err
		}
		cmdio.Log(ctx, progress.NewJobRunUrlEvent(runPageURL(ctx, details.RunPageUrl)))
		return nil, nil
	}

	run, err := waiter.OnProgress(monitor.onProgress).GetWithTimeout(jobRunTimeout)
	if err != nil {
		r.logFailedTasks(ctx, waiter.RunId)
	}
	if err != nil {
		return nil, err
	}
	if run.State.LifeCycleState == jobs.RunLifeCycleStateSkipped {
		log.Infof(ctx, "Run was skipped!")
		return nil, fmt.Errorf("run skipped: %s", run.State.StateMessage)
	}

	switch run.State.ResultState {
	// The run was canceled at user request.
	case jobs.RunResultStateCanceled:
		log.Infof(ctx, "Run was cancelled!")
		return nil, fmt.Errorf("run canceled: %s", run.State.StateMessage)

	// The task completed with an error.
	case jobs.RunResultStateFailed:
		log.Infof(ctx, "Run has failed!")
		return nil, fmt.Errorf("run failed: %s", run.State.StateMessage)

	// The task completed successfully.
	case jobs.RunResultStateSuccess:
		log.Infof(ctx, "Run has completed successfully!")
		return output.GetJobOutput(ctx, r.bundle.WorkspaceClient(ctx), waiter.RunId)

	// The run was stopped after reaching the timeout.
	case jobs.RunResultStateTimedout:
		log.Infof(ctx, "Run has timed out!")
		return nil, fmt.Errorf("run timed out: %s", run.State.StateMessage)

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

	runs, err := w.Jobs.ListRunsAll(ctx, jobs.ListRunsRequest{
		ActiveOnly: true,
		JobId:      jobID,
	})
	if err != nil {
		return err
	}

	if len(runs) == 0 {
		return nil
	}

	errGroup, errCtx := errgroup.WithContext(ctx)
	for _, run := range runs {
		runId := run.RunId
		errGroup.Go(func() error {
			wait, err := w.Jobs.CancelRun(errCtx, jobs.CancelRun{
				RunId: runId,
			})
			if err != nil {
				return err
			}
			// Waits for the Terminated or Skipped state
			_, err = wait.GetWithTimeout(jobRunTimeout)
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
	if continuous != nil && continuous.PauseStatus == jobs.PauseStatusUnpaused {
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
