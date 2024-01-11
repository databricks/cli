package run

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

type jobParameterArgs struct {
	*resources.Job
}

func (a jobParameterArgs) ParseArgs(args []string, opts *Options) error {
	kv, err := genericParseKeyValueArgs(args)
	if err != nil {
		return err
	}

	// Merge the key-value pairs from the args into the options struct.
	if opts.Job.jobParams == nil {
		opts.Job.jobParams = kv
	} else {
		for k, v := range kv {
			opts.Job.jobParams[k] = v
		}
	}
	return nil
}

func (a jobParameterArgs) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string
	for _, param := range a.Parameters {
		completions = append(completions, param.Name)
	}
	return genericCompleteKeyValueArgs(args, toComplete, completions)
}

type jobTaskNotebookParamArgs struct {
	*resources.Job
}

func (a jobTaskNotebookParamArgs) ParseArgs(args []string, opts *Options) error {
	kv, err := genericParseKeyValueArgs(args)
	if err != nil {
		return err
	}

	// Merge the key-value pairs from the args into the options struct.
	if opts.Job.notebookParams == nil {
		opts.Job.notebookParams = kv
	} else {
		for k, v := range kv {
			opts.Job.notebookParams[k] = v
		}
	}
	return nil
}

func (a jobTaskNotebookParamArgs) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	parameters := make(map[string]string)
	for _, t := range a.Tasks {
		if nt := t.NotebookTask; nt != nil {
			maps.Copy(parameters, nt.BaseParameters)
		}
	}
	return genericCompleteKeyValueArgs(args, toComplete, maps.Keys(parameters))
}

type jobTaskJarParamArgs struct {
	*resources.Job
}

func (a jobTaskJarParamArgs) ParseArgs(args []string, opts *Options) error {
	opts.Job.jarParams = append(opts.Job.jarParams, args...)
	return nil
}

func (a jobTaskJarParamArgs) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

type jobTaskPythonParamArgs struct {
	*resources.Job
}

func (a jobTaskPythonParamArgs) ParseArgs(args []string, opts *Options) error {
	opts.Job.pythonParams = append(opts.Job.pythonParams, args...)
	return nil
}

func (a jobTaskPythonParamArgs) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

type jobTaskSparkSubmitParamArgs struct {
	*resources.Job
}

func (a jobTaskSparkSubmitParamArgs) ParseArgs(args []string, opts *Options) error {
	opts.Job.sparkSubmitParams = append(opts.Job.pythonParams, args...)
	return nil
}

func (a jobTaskSparkSubmitParamArgs) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

type jobTaskType int

const (
	jobTaskTypeNotebook jobTaskType = iota + 1
	jobTaskTypeSparkJar
	jobTaskTypeSparkPython
	jobTaskTypeSparkSubmit
	jobTaskTypePipeline
	jobTaskTypePythonWheel
	jobTaskTypeSql
	jobTaskTypeDbt
	jobTaskTypeRunJob
)

func (r *jobRunner) posArgsHandler() argsHandler {
	job := r.job
	if job == nil || job.JobSettings == nil {
		return nopArgsHandler{}
	}

	// Handle job parameters, if any are defined.
	if len(job.Parameters) > 0 {
		return &jobParameterArgs{job}
	}

	// Handle task parameters otherwise.
	var seen = make(map[jobTaskType]bool)
	var typ jobTaskType
	for _, t := range job.Tasks {
		if t.NotebookTask != nil {
			typ = jobTaskTypeNotebook
			seen[typ] = true
		}
		if t.SparkJarTask != nil {
			typ = jobTaskTypeSparkJar
			seen[typ] = true
		}
		if t.SparkPythonTask != nil {
			typ = jobTaskTypeSparkPython
			seen[typ] = true
		}
		if t.SparkSubmitTask != nil {
			typ = jobTaskTypeSparkSubmit
			seen[typ] = true
		}
		if t.PipelineTask != nil {
			typ = jobTaskTypePipeline
			seen[typ] = true
		}
		if t.PythonWheelTask != nil {
			typ = jobTaskTypePythonWheel
			seen[typ] = true
		}
		if t.SqlTask != nil {
			typ = jobTaskTypeSql
			seen[typ] = true
		}
		if t.DbtTask != nil {
			typ = jobTaskTypeDbt
			seen[typ] = true
		}
		if t.RunJobTask != nil {
			typ = jobTaskTypeRunJob
			seen[typ] = true
		}
	}

	// Cannot handle positional arguments if we have more than one task type.
	if len(seen) != 1 {
		return nopArgsHandler{}
	}

	switch typ {
	case jobTaskTypeNotebook:
		return jobTaskNotebookParamArgs{job}
	case jobTaskTypeSparkJar:
		return jobTaskJarParamArgs{job}
	case jobTaskTypeSparkPython, jobTaskTypePythonWheel:
		return jobTaskPythonParamArgs{job}
	case jobTaskTypeSparkSubmit:
		return jobTaskSparkSubmitParamArgs{job}
	default:
		// No positional argument handling for other task types.
		return nopArgsHandler{}
	}
}
