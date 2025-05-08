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
	opts.Job.sparkSubmitParams = append(opts.Job.sparkSubmitParams, args...)
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
	if job == nil {
		return nopArgsHandler{}
	}

	// Handle job parameters, if any are defined.
	if len(job.Parameters) > 0 {
		return &jobParameterArgs{job}
	}

	// Handle task parameters otherwise.
	seen := make(map[jobTaskType]bool)
	for _, t := range job.Tasks {
		if t.NotebookTask != nil {
			seen[jobTaskTypeNotebook] = true
		}
		if t.SparkJarTask != nil {
			seen[jobTaskTypeSparkJar] = true
		}
		if t.SparkPythonTask != nil {
			seen[jobTaskTypeSparkPython] = true
		}
		if t.SparkSubmitTask != nil {
			seen[jobTaskTypeSparkSubmit] = true
		}
		if t.PipelineTask != nil {
			seen[jobTaskTypePipeline] = true
		}
		if t.PythonWheelTask != nil {
			seen[jobTaskTypePythonWheel] = true
		}
		if t.SqlTask != nil {
			seen[jobTaskTypeSql] = true
		}
		if t.DbtTask != nil {
			seen[jobTaskTypeDbt] = true
		}
		if t.RunJobTask != nil {
			seen[jobTaskTypeRunJob] = true
		}
	}

	// Cannot handle positional arguments if we have more than one task type.
	keys := maps.Keys(seen)
	if len(keys) != 1 {
		return nopArgsHandler{}
	}

	switch keys[0] {
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
