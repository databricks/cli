package run

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/fatih/color"
	flag "github.com/spf13/pflag"
)

// JobOptions defines options for running a job.
type JobOptions struct {
	dbtCommands       []string
	jarParams         []string
	notebookParams    map[string]string
	pipelineParams    map[string]string
	pythonNamedParams map[string]string
	pythonParams      []string
	sparkSubmitParams []string
	sqlParams         map[string]string
}

func (o *JobOptions) Define(fs *flag.FlagSet) {
	fs.StringSliceVar(&o.dbtCommands, "dbt-commands", nil, "A list of commands to execute for jobs with DBT tasks.")
	fs.StringSliceVar(&o.jarParams, "jar-params", nil, "A list of parameters for jobs with Spark JAR tasks.")
	fs.StringToStringVar(&o.notebookParams, "notebook-params", nil, "A map from keys to values for jobs with notebook tasks.")
	fs.StringToStringVar(&o.pipelineParams, "pipeline-params", nil, "A map from keys to values for jobs with pipeline tasks.")
	fs.StringToStringVar(&o.pythonNamedParams, "python-named-params", nil, "A map from keys to values for jobs with Python wheel tasks.")
	fs.StringSliceVar(&o.pythonParams, "python-params", nil, "A list of parameters for jobs with Python tasks.")
	fs.StringSliceVar(&o.sparkSubmitParams, "spark-submit-params", nil, "A list of parameters for jobs with Spark submit tasks.")
	fs.StringToStringVar(&o.sqlParams, "sql-params", nil, "A map from keys to values for jobs with SQL tasks.")
}

func (o *JobOptions) validatePipelineParams() (*jobs.PipelineParams, error) {
	if len(o.pipelineParams) == 0 {
		return nil, nil
	}

	var defaultErr = fmt.Errorf("job run argument --pipeline-params only supports `full_refresh=<bool>`")
	v, ok := o.pipelineParams["full_refresh"]
	if !ok {
		return nil, defaultErr
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil, defaultErr
	}

	pipelineParams := &jobs.PipelineParams{
		FullRefresh: b,
	}

	return pipelineParams, nil
}

func (o *JobOptions) toPayload(jobID int64) (*jobs.RunNow, error) {
	pipelineParams, err := o.validatePipelineParams()
	if err != nil {
		return nil, err
	}

	payload := &jobs.RunNow{
		JobId: jobID,

		DbtCommands:       o.dbtCommands,
		JarParams:         o.jarParams,
		NotebookParams:    o.notebookParams,
		PipelineParams:    pipelineParams,
		PythonNamedParams: o.pythonNamedParams,
		PythonParams:      o.pythonParams,
		SparkSubmitParams: o.sparkSubmitParams,
		SqlParams:         o.sqlParams,
	}

	return payload, nil
}

// Default timeout for waiting for a job run to complete.
var jobRunTimeout time.Duration = 2 * time.Hour

type jobRunner struct {
	key

	bundle *bundle.Bundle
	job    *resources.Job
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

func (r *jobRunner) logRun(ctx context.Context, runId int64) {
	w := r.bundle.WorkspaceClient()
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	var errorPrefix = fmt.Sprintf("%s [%s]", red("[ERROR]"), r.Key())
	var infoPrefix = fmt.Sprintf("%s [%s]", "[INFO]", r.Key())
	run, err := w.Jobs.GetRun(ctx, jobs.GetRun{
		RunId: runId,
	})
	if err != nil {
		log.Printf("%s failed to log job run. Error: %s", errorPrefix, err)
		return
	}
	if run.State.ResultState == jobs.RunResultStateSuccess {
		log.Printf("%s all tasks executed successfully", infoPrefix)
		return
	}
	for _, task := range run.Tasks {
		if isSuccess(task) {
			log.Printf("%s task %s completed successfully", infoPrefix, green(task.TaskKey))
		} else if isFailed(task) {
			taskInfo, err := w.Jobs.GetRunOutput(ctx, jobs.GetRunOutput{
				RunId: task.RunId,
			})
			if err != nil {
				log.Printf("%s task %s failed. Unable to fetch error trace: %s",
					errorPrefix, red(task.TaskKey), err)
				return
			}
			log.Printf("%s Task %s failed!\nError:\n%s\nTrace:\n%s", errorPrefix,
				red(task.TaskKey), taskInfo.Error, taskInfo.ErrorTrace)
		} else {
			log.Printf("%s task %s is in state %s", infoPrefix,
				yellow(task.TaskKey), task.State.LifeCycleState)
		}
	}

}

func (r *jobRunner) Run(ctx context.Context, opts *Options) error {
	jobID, err := strconv.ParseInt(r.job.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("job ID is not an integer: %s", r.job.ID)
	}

	var prefix = fmt.Sprintf("[INFO] [%s]", r.Key())
	var prevState *jobs.RunState
	var runId *int64

	// This function is called each time the function below polls the run status.
	update := func(info *retries.Info[jobs.Run]) {
		i := info.Info
		if i == nil {
			return
		}

		state := i.State
		if state == nil {
			return
		}

		// Log the job run URL as soon as it is available.
		if prevState == nil {
			log.Printf("%s Run available at %s", prefix, info.Info.RunPageUrl)
		}
		if prevState == nil || prevState.LifeCycleState != state.LifeCycleState {
			log.Printf("%s Run status: %s", prefix, info.Info.State.LifeCycleState)
			prevState = state
		}
		if runId == nil {
			runId = &i.RunId
		}
	}

	req, err := opts.Job.toPayload(jobID)
	if err != nil {
		return err
	}

	w := r.bundle.WorkspaceClient()

	run, err := w.Jobs.RunNowAndWait(ctx, *req, retries.Timeout[jobs.Run](jobRunTimeout), update)
	if runId == nil {
		log.Printf("%s runId unavailable. Skipping logging job run", prefix)
	} else {
		r.logRun(ctx, *runId)

	}
	if err != nil {
		return err
	}

	switch run.State.ResultState {
	// The run was canceled at user request.
	case jobs.RunResultStateCanceled:
		log.Printf("%s Run was cancelled!", prefix)
		return fmt.Errorf("run canceled: %s", run.State.StateMessage)

	// The task completed with an error.
	case jobs.RunResultStateFailed:
		log.Printf("%s Run has failed!", prefix)
		return fmt.Errorf("run failed: %s", run.State.StateMessage)

	// The task completed successfully.
	case jobs.RunResultStateSuccess:
		log.Printf("%s Run has completed successfully!", prefix)
		return nil

	// The run was stopped after reaching the timeout.
	case jobs.RunResultStateTimedout:
		log.Printf("%s Run has timed out!", prefix)
		return fmt.Errorf("run timed out: %s", run.State.StateMessage)
	}

	return err
}
