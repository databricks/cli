package run

import (
	"errors"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	flag "github.com/spf13/pflag"
)

// JobOptions defines options for running a job.
type JobOptions struct {
	// Task parameters are specific to the type of task.
	dbtCommands       []string
	jarParams         []string
	notebookParams    map[string]string
	pipelineParams    map[string]string
	pythonNamedParams map[string]string
	pythonParams      []string
	sparkSubmitParams []string
	sqlParams         map[string]string

	// Job parameters are a map of key-value pairs that are passed to the job.
	// If a job uses job parameters, it cannot use task parameters.
	// Also see https://docs.databricks.com/en/workflows/jobs/settings.html#add-parameters-for-all-job-tasks.
	jobParams map[string]string
}

func (o *JobOptions) DefineJobOptions(fs *flag.FlagSet) {
	fs.StringToStringVar(&o.jobParams, "params", nil, "comma separated k=v pairs for job parameters")
}

func (o *JobOptions) DefineTaskOptions(fs *flag.FlagSet) {
	fs.StringSliceVar(&o.dbtCommands, "dbt-commands", nil, "A list of commands to execute for jobs with DBT tasks.")
	fs.StringSliceVar(&o.jarParams, "jar-params", nil, "A list of parameters for jobs with Spark JAR tasks.")
	fs.StringToStringVar(&o.notebookParams, "notebook-params", nil, "A map from keys to values for jobs with notebook tasks.")
	fs.StringToStringVar(&o.pipelineParams, "pipeline-params", nil, "A map from keys to values for jobs with pipeline tasks.")
	fs.StringToStringVar(&o.pythonNamedParams, "python-named-params", nil, "A map from keys to values for jobs with Python wheel tasks.")
	fs.StringSliceVar(&o.pythonParams, "python-params", nil, "A list of parameters for jobs with Python tasks.")
	fs.StringSliceVar(&o.sparkSubmitParams, "spark-submit-params", nil, "A list of parameters for jobs with Spark submit tasks.")
	fs.StringToStringVar(&o.sqlParams, "sql-params", nil, "A map from keys to values for jobs with SQL tasks.")
}

func (o *JobOptions) hasTaskParametersConfigured() bool {
	return len(o.dbtCommands) > 0 ||
		len(o.jarParams) > 0 ||
		len(o.notebookParams) > 0 ||
		len(o.pipelineParams) > 0 ||
		len(o.pythonNamedParams) > 0 ||
		len(o.pythonParams) > 0 ||
		len(o.sparkSubmitParams) > 0 ||
		len(o.sqlParams) > 0
}

func (o *JobOptions) hasJobParametersConfigured() bool {
	return len(o.jobParams) > 0
}

// Validate returns if the combination of options is valid.
func (o *JobOptions) Validate(job *resources.Job) error {
	if job == nil {
		return errors.New("job not defined")
	}

	// Ensure mutual exclusion on job parameters and task parameters.
	hasJobParams := len(job.Parameters) > 0
	if hasJobParams && o.hasTaskParametersConfigured() {
		return errors.New("the job to run defines job parameters; specifying task parameters is not allowed")
	}
	if !hasJobParams && o.hasJobParametersConfigured() {
		return errors.New("the job to run does not define job parameters; specifying job parameters is not allowed")
	}

	return nil
}

func (o *JobOptions) validatePipelineParams() (*jobs.PipelineParams, error) {
	if len(o.pipelineParams) == 0 {
		return nil, nil
	}

	defaultErr := errors.New("job run argument --pipeline-params only supports `full_refresh=<bool>`")
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

func (o *JobOptions) toPayload(job *resources.Job, jobID int64) (*jobs.RunNow, error) {
	if err := o.Validate(job); err != nil {
		return nil, err
	}

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

		JobParameters: o.jobParams,
	}

	return payload, nil
}
