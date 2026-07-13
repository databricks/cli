package trainyaml

import (
	"errors"
	"fmt"
	"path"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// environmentKey is the job environment key the generated AI Runtime task
// references. AI Runtime tasks require a serverless environment; we materialize
// one visibly in the generated config so the user can edit it afterwards.
const environmentKey = "default"

// commandScriptName is the file the inline train.yaml command is written to.
// The AI Runtime task's command_path points at it, relative to the code source.
const commandScriptName = "command.sh"

// Result holds everything Convert produces from a train.yaml: the DABs job
// resource, the shell script the command was materialized into, the artifact
// path (set only when the snapshot targets a UC volume), and any warnings about
// train.yaml fields that could not be represented.
type Result struct {
	Job           *resources.Job
	CommandScript string
	ArtifactPath  string
	Warnings      []string
}

// Convert maps a parsed train.yaml Config to a DABs job resource. It returns an
// error for inputs that cannot be represented as an AI Runtime task (legacy
// accelerators, pools, docker images) and collects warnings for fields that are
// dropped because they have no equivalent.
func Convert(cfg *Config) (*Result, error) {
	if cfg.ExperimentName == "" {
		return nil, errors.New("experiment_name is required")
	}
	if cfg.Command == "" {
		return nil, errors.New("command is required")
	}
	if cfg.CodeSource == nil || cfg.CodeSource.Snapshot == nil || cfg.CodeSource.Snapshot.RootPath == "" {
		return nil, errors.New("code_source.snapshot.root_path is required to migrate to an AI Runtime task")
	}

	accelerator, err := convertAcceleratorType(cfg.Compute.AcceleratorType)
	if err != nil {
		return nil, err
	}
	if cfg.Compute.NodePoolID != "" || cfg.Compute.PoolName != "" {
		return nil, errors.New("compute pools (node_pool_id/pool_name) are not supported by ai_runtime_task; remove them to use serverless AI Runtime compute")
	}

	env, err := convertEnvironment(cfg.Environment)
	if err != nil {
		return nil, err
	}

	res := &Result{}

	task := jobs.Task{
		TaskKey:        cfg.ExperimentName,
		EnvironmentKey: environmentKey,
		AiRuntimeTask: &jobs.AiRuntimeTask{
			Experiment:                cfg.ExperimentName,
			MlflowRun:                 cfg.MlflowRunName,
			MlflowExperimentDirectory: cfg.MlflowExperimentDirectory,
			CodeSourcePath:            cfg.CodeSource.Snapshot.RootPath,
			Deployments: []jobs.DeploymentSpec{
				{
					// command_path is relative to the extracted code source root.
					CommandPath: path.Join(path.Base(cfg.CodeSource.Snapshot.RootPath), commandScriptName),
					Compute: jobs.ComputeSpec{
						AcceleratorType:  accelerator,
						AcceleratorCount: cfg.Compute.NumAccelerators,
					},
				},
			},
		},
	}
	if cfg.MaxRetries != nil {
		task.MaxRetries = *cfg.MaxRetries
	}
	if cfg.TimeoutMinutes != nil {
		task.TimeoutSeconds = *cfg.TimeoutMinutes * 60
	}

	res.Job = &resources.Job{
		JobSettings: jobs.JobSettings{
			Name:         cfg.ExperimentName,
			Tasks:        []jobs.Task{task},
			Environments: []jobs.JobEnvironment{{EnvironmentKey: environmentKey, Spec: env}},
		},
	}

	res.CommandScript = cfg.Command
	if cfg.CodeSource.Snapshot.RemoteVolume != "" {
		res.ArtifactPath = cfg.CodeSource.Snapshot.RemoteVolume
	}

	res.Warnings = append(res.Warnings, unmappedWarnings(cfg)...)
	return res, nil
}

func convertAcceleratorType(v string) (jobs.ComputeSpecAcceleratorType, error) {
	if v == "" {
		return "", errors.New("compute.accelerator_type is required")
	}
	var t jobs.ComputeSpecAcceleratorType
	if err := t.Set(v); err != nil {
		return "", fmt.Errorf("compute.accelerator_type %q is not supported by ai_runtime_task (expected GPU_1xA10, GPU_1xH100, or GPU_8xH100)", v)
	}
	return t, nil
}

func convertEnvironment(env *Environment) (*compute.Environment, error) {
	if env == nil {
		return &compute.Environment{}, nil
	}
	if env.DockerImage != nil {
		return nil, errors.New("environment.docker_image is not supported by ai_runtime_task; use environment.version and environment.dependencies instead")
	}
	if env.Dependencies.Path != "" {
		return nil, errors.New("environment.dependencies as a requirements file path is not supported; provide an inline list of dependencies instead")
	}

	return &compute.Environment{
		EnvironmentVersion: env.Version,
		Dependencies:       env.Dependencies.List,
	}, nil
}

// unmappedWarnings reports train.yaml fields that have no ai_runtime_task
// equivalent and are therefore dropped from the generated config.
func unmappedWarnings(cfg *Config) []string {
	var warnings []string
	warn := func(field, hint string) {
		warnings = append(warnings, fmt.Sprintf("%q has no ai_runtime_task equivalent and was dropped%s", field, hint))
	}

	if len(cfg.Secrets) > 0 {
		warn("secrets", "; configure secrets on the job or via {{secrets/scope/key}} references")
	}
	if len(cfg.EnvVariables) > 0 {
		warn("env_variables", "; set them inside command.sh or the serverless environment")
	}
	if len(cfg.Parameters) > 0 {
		warn("parameters", "")
	}
	if cfg.Priority != nil {
		warn("priority", "")
	}
	if cfg.UsagePolicyName != "" {
		warn("usage_policy_name", "")
	}
	if cfg.UsagePolicyID != "" {
		warn("usage_policy_id", "")
	}
	if cfg.IdempotencyToken != "" {
		warn("idempotency_token", "")
	}
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil && len(cfg.CodeSource.Snapshot.IncludePaths) > 0 {
		warn("code_source.snapshot.include_paths", "; the entire code_source directory is packaged")
	}
	return warnings
}
