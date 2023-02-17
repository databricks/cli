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
	flag "github.com/spf13/pflag"
)

func colorRed(s string) string {
	const colorReset = "\033[0m"
	const colorRed = "\033[31m"
	return colorRed + s + colorReset
}

func colorGreen(s string) string {
	const colorReset = "\033[0m"
	const colorGreen = "\033[32m"
	return colorGreen + s + colorReset
}

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

func (r *jobRunner) Run(ctx context.Context, opts *Options) error {
	jobID, err := strconv.ParseInt(r.job.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("job ID is not an integer: %s", r.job.ID)
	}

	var prefix = fmt.Sprintf("[INFO] [%s]", r.Key())
	var errorPrefix = fmt.Sprintf("%s [%s]", colorRed("[ERROR]"), r.Key())
	var prevState *jobs.RunState

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
	}

	req, err := opts.Job.toPayload(jobID)
	if err != nil {
		return err
	}

	w := r.bundle.WorkspaceClient()

	// TODO: we will need to get run ID for this run. To do that either
	// 1. Return runId on error from the SDK. Might not be trivial since the file seems autogenerate
	// 2. Add a new custom method in the SDK to wait for a run to terminate
	//
	// For now proceeding with modifying the sdk to move onto figuring out how to render the
	// error
	run, err := w.Jobs.RunNowAndWait(ctx, *req, retries.Timeout[jobs.Run](jobRunTimeout), update)
	// runJson, _ := json.MarshalIndent(run, "", " ")
	// fmt.Println("AAAA export out json: \n", string(runJson))

	for _, task := range run.Tasks {
		// fmt.Printf("task %s state is %s.\n", task.TaskKey, task.State.ResultState)
		if task.State.LifeCycleState == jobs.RunLifeCycleStateInternalError || task.State.ResultState == jobs.RunResultStateFailed {
			runOutput, err2 := w.Jobs.GetRunOutput(ctx, jobs.GetRunOutput{
				RunId: task.RunId,
			})
			if err2 != nil {
				return err2
			}
			log.Printf("%s Task %s failed!\n%s\nTrace:\n%s", errorPrefix, colorRed(task.TaskKey), runOutput.Error, runOutput.ErrorTrace)
		}
		if task.State.ResultState == jobs.RunResultStateSuccess {
			log.Printf("%s Task %s succeeded.", prefix, colorGreen(task.TaskKey))
		}
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
