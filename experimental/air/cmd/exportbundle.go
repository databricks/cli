package aircmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// This file implements the train.yaml -> databricks.yml conversion behind
// `air export-bundle`: a one-time converter that turns an AIR run YAML into a
// Databricks Asset Bundle deploying the same workload as a native ai_runtime_task.
// It is the durable counterpart to `air run` (which submits an ephemeral one-time
// run): the emitted bundle is a persistent Jobs resource the user owns and deploys
// with `databricks bundle deploy`.
//
// The field mapping mirrors buildSubmitPayload (runsubmit.go) so the generated
// task is the shape `air run` would submit. Structural validity is already
// guaranteed upstream by runConfig.validate(); this file adds a second,
// converter-specific gate — checkBundleConvertible — that rejects the configs a
// bundle cannot represent faithfully, rather than emitting a databricks.yml that
// deploys but misbehaves. The gate exists because train.yaml and a bundle are not
// 1:1: some run fields have no bundle equivalent, and some assume the AIR run
// harness (e.g. $CODE_SOURCE_PATH) that a bundle does not provide.

// bundleCommandScript is the entrypoint filename the emitted bundle references and
// that `bundle sync` uploads alongside the user's code.
const bundleCommandScript = "command.sh"

// codeSourcePathVar is the environment variable the AIR run harness sets to the
// extracted snapshot directory. A bundle delivers code via `bundle sync` and never
// sets it, so a command relying on it would break at runtime — the convertibility
// gate rejects such commands.
const codeSourcePathVar = "$CODE_SOURCE_PATH"

// workspaceFilePathRef is the bundle variable that resolves to where `bundle
// deploy` syncs this folder; the emitted command_path points under it.
const workspaceFilePathRef = "${workspace.file_path}"

// aiRuntimeEnvVarsKey is the key linking the task's environment_variables_key to
// the single job-level env-var profile the converter emits. Mirrors
// AI_RUNTIME_ENV_VARS_KEY in the Python CLI (common Jobs env-var API, API-293).
const aiRuntimeEnvVarsKey = "default"

// checkBundleConvertible reports why a structurally-valid runConfig cannot be
// converted to a faithful bundle, or nil if it can. Every reason names the source
// field and why it can't be represented, so the CLI can reject with an actionable
// message instead of emitting a lossy databricks.yml.
func checkBundleConvertible(cfg *runConfig) error {
	var reasons []string

	// env_variables / secrets ARE now representable: they ride the common Jobs
	// env-var API (API-293) as a job-level environment_variables profile the task
	// references by key. convertToBundle emits that profile, so no rejection here.
	// (Verified on staging: the profile persists on runs/get; secrets are emitted
	// as {{secrets/scope/key}} refs resolved by Jobs at run time.)

	// docker_image: a custom image must be registered before it can be referenced;
	// that registration is not part of a bundle deploy.
	if cfg.dockerImageURL() != "" {
		reasons = append(reasons, "environment.docker_image: needs image registration, which a bundle deploy does not perform")
	}

	// usage_policy_*: the name/id resolves to a budget policy via a workspace
	// lookup at submit time; a static bundle has nowhere to run that lookup.
	if cfg.UsagePolicyName != nil {
		reasons = append(reasons, "usage_policy_name: resolved by a workspace lookup at submit time, not representable statically")
	}
	if cfg.UsagePolicyID != nil {
		reasons = append(reasons, "usage_policy_id: budget policy binding is not represented on the ai_runtime_task bundle path yet")
	}

	// code_source with a git ref: bundle sync uploads the working tree; it cannot
	// pin to a specific commit or fetch a remote branch (the R1/R3 gap in the
	// uploads-vs-DABs design doc). Dropping the pin would silently change what runs.
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil && cfg.CodeSource.Snapshot.Git != nil {
		reasons = append(reasons, "code_source.snapshot.git: bundle sync cannot pin to a git commit or fetch a remote branch")
	}

	// A command that reads $CODE_SOURCE_PATH assumes the AIR run harness, which a
	// bundle does not provide; the synced code lives under ${workspace.file_path}.
	if cfg.Command != nil && strings.Contains(*cfg.Command, codeSourcePathVar) {
		reasons = append(reasons, fmt.Sprintf(
			"command references %s, which only exists on the `air run` path; a bundle syncs code to %s instead",
			codeSourcePathVar, workspaceFilePathRef))
	}

	if len(reasons) == 0 {
		return nil
	}
	return fmt.Errorf(
		"this train.yaml cannot be converted to a faithful bundle:\n  - %s\nrun it with `air run` until these are supported on the bundle path",
		strings.Join(reasons, "\n  - "))
}

// exportedBundle is the minimal databricks.yml shape the converter emits: a bundle
// name plus one job with a single ai_runtime_task. It marshals to YAML, so field
// order here is the emitted key order.
type exportedBundle struct {
	Bundle    bundleBlock            `yaml:"bundle"`
	Resources exportedResourcesBlock `yaml:"resources"`
}

type bundleBlock struct {
	Name string `yaml:"name"`
}

type exportedResourcesBlock struct {
	Jobs map[string]exportedJob `yaml:"jobs"`
}

type exportedJob struct {
	Name         string                `yaml:"name"`
	Tasks        []exportedTask        `yaml:"tasks"`
	Environments []exportedEnvironment `yaml:"environments"`
	// EnvironmentVariables carries env-var profiles via the common Jobs env-var API
	// (API-293). Emitted only when the run declares env_variables/secrets; the task
	// references a profile by key (see exportedTask.EnvironmentVariablesKey).
	EnvironmentVariables []exportedEnvVarProfile `yaml:"environment_variables,omitempty"`
}

type exportedTask struct {
	TaskKey        string `yaml:"task_key"`
	EnvironmentKey string `yaml:"environment_key"`
	// EnvironmentVariablesKey references a job-level environment_variables profile
	// (common Jobs env-var API). Omitted when the run has no env vars/secrets so the
	// wire form matches the submit path, which sets it only under the same gate.
	EnvironmentVariablesKey string                `yaml:"environment_variables_key,omitempty"`
	MaxRetries              int                   `yaml:"max_retries"`
	TimeoutSeconds          int                   `yaml:"timeout_seconds,omitempty"`
	AiRuntimeTask           exportedAiRuntimeTask `yaml:"ai_runtime_task"`
}

// exportedEnvVarProfile is one entry in the job-level environment_variables list
// (common Jobs env-var API). variables holds plain values inline and secrets as
// {{secrets/scope/key}} references, resolved by Jobs at run time — the exact shape
// the ai_runtime_task submit path emits (jobs_api_client.py, API-293).
type exportedEnvVarProfile struct {
	EnvironmentVariablesKey string            `yaml:"environment_variables_key"`
	Variables               map[string]string `yaml:"variables"`
}

type exportedAiRuntimeTask struct {
	Experiment                string               `yaml:"experiment"`
	MlflowRun                 string               `yaml:"mlflow_run,omitempty"`
	MlflowExperimentDirectory string               `yaml:"mlflow_experiment_directory,omitempty"`
	Deployments               []exportedDeployment `yaml:"deployments"`
}

type exportedDeployment struct {
	Name        string          `yaml:"name"`
	CommandPath string          `yaml:"command_path"`
	Compute     exportedCompute `yaml:"compute"`
}

type exportedCompute struct {
	AcceleratorType  string `yaml:"accelerator_type"`
	AcceleratorCount int    `yaml:"accelerator_count"`
}

type exportedEnvironment struct {
	EnvironmentKey string          `yaml:"environment_key"`
	Spec           exportedEnvSpec `yaml:"spec"`
}

// exportedEnvSpec carries the serverless runtime selection. The two fields are
// mutually exclusive (the Jobs serverless validator rejects setting both): a bare
// numeric channel uses environment_version; the databricks-ai managed environment
// (torch + ML venv preinstalled) uses base_environment. omitempty on both so only
// the resolved one is emitted.
type exportedEnvSpec struct {
	EnvironmentVersion string `yaml:"environment_version,omitempty"`
	BaseEnvironment    string `yaml:"base_environment,omitempty"`
}

// databricksAITokenPrefix marks an environment.version that selects the
// databricks-ai managed base environment rather than a bare channel. Mirrors
// _DATABRICKS_AI_TOKEN_PREFIX in the Python CLI's jobs_api_client.py.
const databricksAITokenPrefix = "databricks_ai_v"

// databricksAIBaseEnvironment is the system base_environment id the databricks-ai
// token resolves to. Mirrors _DATABRICKS_AI_BASE_ENVIRONMENT in the Python CLI.
const databricksAIBaseEnvironment = "workspace-base-environments/"

// convertToBundle maps a convertible runConfig to the emitted bundle. It assumes
// checkBundleConvertible has already passed. The command_path points at the synced
// command.sh under ${workspace.file_path}; the runtime channel is resolved the same
// way `air run` resolves it (config version, else the default), minus the process
// env lookup so a generated artifact is reproducible rather than host-dependent.
func convertToBundle(cfg *runConfig) *exportedBundle {
	task := exportedAiRuntimeTask{
		Experiment: cfg.ExperimentName,
		Deployments: []exportedDeployment{{
			// `air run` submits a single unnamed deployment; the bundle names it
			// "worker" so the authored YAML reads clearly. The name is cosmetic (the
			// submit payload carries none), so this does not change behavior.
			Name:        "worker",
			CommandPath: workspaceFilePathRef + "/" + bundleCommandScript,
			Compute: exportedCompute{
				AcceleratorType:  cfg.Compute.AcceleratorType,
				AcceleratorCount: cfg.Compute.NumAccelerators,
			},
		}},
	}
	if cfg.MLflowRunName != nil {
		task.MlflowRun = *cfg.MLflowRunName
	}
	if cfg.MLflowExperimentDirectory != nil {
		task.MlflowExperimentDirectory = *cfg.MLflowExperimentDirectory
	}

	tsk := exportedTask{
		TaskKey:        cfg.ExperimentName,
		EnvironmentKey: aiRuntimeEnvironmentKey,
		MaxRetries:     cfg.maxRetries(),
		TimeoutSeconds: cfg.timeoutSeconds(),
		AiRuntimeTask:  task,
	}

	// Env vars ride the common Jobs env-var API: one job-level profile, referenced
	// by the task's environment_variables_key. Emitted only when there are vars or
	// secrets, so a var-free run's wire form matches the submit path exactly.
	profiles := envVarProfiles(cfg)
	if len(profiles) > 0 {
		tsk.EnvironmentVariablesKey = aiRuntimeEnvVarsKey
	}

	// Resource key shares the task_key charset, which validateExperimentName
	// already guarantees, so the experiment name is safe to use verbatim.
	return &exportedBundle{
		Bundle: bundleBlock{Name: cfg.ExperimentName},
		Resources: exportedResourcesBlock{
			Jobs: map[string]exportedJob{
				cfg.ExperimentName: {
					Name:  cfg.ExperimentName,
					Tasks: []exportedTask{tsk},
					Environments: []exportedEnvironment{{
						EnvironmentKey: aiRuntimeEnvironmentKey,
						Spec:           exportBundleEnvSpec(cfg),
					}},
					EnvironmentVariables: profiles,
				},
			},
		},
	}
}

// envVarProfiles builds the job-level env-var profile list from the run's
// env_variables and secrets, or nil when there are none. Plain values are inline;
// each secret (ENV_VAR -> "scope/key") becomes a {{secrets/scope/key}} reference
// Jobs resolves at run time. This mirrors the Python CLI's ai_runtime_task path
// (jobs_api_client.py, API-293) and was verified against staging: the profile
// persists on runs/get and secret refs resolve at run time.
func envVarProfiles(cfg *runConfig) []exportedEnvVarProfile {
	if len(cfg.EnvVariables) == 0 && len(cfg.Secrets) == 0 {
		return nil
	}
	variables := make(map[string]string, len(cfg.EnvVariables)+len(cfg.Secrets))
	for k, v := range cfg.EnvVariables {
		variables[k] = v
	}
	for envVar, secretRef := range cfg.Secrets {
		variables[envVar] = "{{secrets/" + secretRef + "}}"
	}
	return []exportedEnvVarProfile{{
		EnvironmentVariablesKey: aiRuntimeEnvVarsKey,
		Variables:               variables,
	}}
}

// exportBundleEnvSpec resolves the serverless runtime selection for the bundle,
// branching on environment.version the same way the Python CLI's jobs_api_client
// does (jobs_api_client.py:1562-1565): a "databricks_ai_v<N>" token selects the
// managed databricks-ai base_environment (torch + ML venv), while a bare numeric
// channel ("4", "5", ...) uses environment_version. Unlike dlRuntimeImage it does
// not read process env, so the generated bundle is reproducible. A requirements-file
// dependency set carries its version in the file, so this falls back to the default
// channel there (same as the submit path).
//
// TODO(air): the Go `air run` submit path (runsubmit.go) only ever emits
// environment_version and cannot select base_environment, so a `version:
// databricks_ai_v<N>` run lands on a bare GPU channel without torch/mlflow and
// fails at import. This converter hardcodes the correct base_environment branch so
// exported bundles run today; once `air run` forwards env vars/dependencies through
// the new BYOT common Jobs API (env_file/env_var work, ETA EOQ2), the submit path
// and this converter should share one runtime-selection helper instead of
// duplicating the branch.
func exportBundleEnvSpec(cfg *runConfig) exportedEnvSpec {
	channel := strings.TrimPrefix(defaultDlRuntimeImage, "CLIENT-GPU-")
	if v, ok := cfg.runtimeVersion(); ok {
		channel = strings.TrimPrefix(v, "CLIENT-GPU-")
	}
	if strings.HasPrefix(channel, databricksAITokenPrefix) {
		return exportedEnvSpec{BaseEnvironment: databricksAIBaseEnvironment + channel}
	}
	return exportedEnvSpec{EnvironmentVersion: channel}
}

// marshalBundle renders the bundle to YAML with a header explaining provenance and
// the steps the user must complete before deploying (add a targets block; the code
// and command.sh are synced from the bundle folder).
func marshalBundle(b *exportedBundle, sourcePath string) ([]byte, error) {
	body, err := yaml.Marshal(b)
	if err != nil {
		return nil, err
	}
	header := "# Generated by `air export-bundle` from " + filepath.Base(sourcePath) + ".\n" +
		"#\n" +
		"# Deploys the same workload as a durable Jobs resource. `bundle deploy` syncs this\n" +
		"# folder (including " + bundleCommandScript + " and your code) to the workspace; the task's\n" +
		"# command_path points at the synced " + bundleCommandScript + ". Before deploying, add a\n" +
		"# `targets` block with your workspace host. See the ai-compute DABs examples.\n"
	return append([]byte(header), body...), nil
}
