package aircmd

import (
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// This file converts a train.yaml runConfig to a databricks.yml Asset Bundle
// deploying the same workload as a native ai_runtime_task. It backs both
// `air export-bundle` and the `air run` submit path (rundabs.go).
//
// checkBundleConvertible rejects configs a bundle cannot represent faithfully:
// train.yaml and a bundle are not 1:1, and some run fields have no bundle
// equivalent or assume the AIR run harness (e.g. $CODE_SOURCE_PATH) a bundle does
// not provide.

// bundleCommandScript is the entrypoint filename the emitted bundle references and
// that `bundle sync` uploads alongside the user's code.
const bundleCommandScript = "command.sh"

// codeSourcePathVar is the environment variable the AIR run harness sets to the
// extracted snapshot directory. A bundle delivers code via `bundle sync` and never
// sets it, so the gate rejects commands relying on it.
const codeSourcePathVar = "$CODE_SOURCE_PATH"

// workspaceFilePathRef is the bundle variable that resolves to where `bundle
// deploy` syncs this folder; the emitted command_path points under it.
const workspaceFilePathRef = "${workspace.file_path}"

// aiRuntimeEnvVarsKey links the task's environment_variables_key to the single
// job-level env-var profile the converter emits.
const aiRuntimeEnvVarsKey = "default"

// checkBundleConvertible reports why a structurally-valid runConfig cannot be
// converted to a faithful bundle, or nil if it can. Each reason names the source
// field so the CLI can reject with an actionable message rather than emit a lossy
// databricks.yml. env_variables/secrets are representable (see envVarProfiles) and
// are not rejected here.
func checkBundleConvertible(cfg *runConfig) error {
	var reasons []string

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

	// code_source snapshots are delivered by uploading the working tree as an
	// immutable-folder snapshot (see writeBundleProject). Two sub-cases can't be
	// represented that way:
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil {
		snap := cfg.CodeSource.Snapshot
		// git ref: the snapshot uploads the live working tree; it cannot pin to a
		// commit or fetch a remote branch. Dropping the pin would change what runs.
		if snap.Git != nil {
			reasons = append(reasons, "code_source.snapshot.git: the immutable-folder snapshot uploads the working tree and cannot pin a git commit or fetch a remote branch")
		}
		// remote_volume: the immutable-folder snapshot uploads to Workspace Files
		// only, not a UC Volume.
		if snap.RemoteVolume != nil {
			reasons = append(reasons, "code_source.snapshot.remote_volume: the immutable-folder snapshot uploads to Workspace Files, not a UC Volume")
		}
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
	Bundle       bundleBlock            `yaml:"bundle"`
	Resources    exportedResourcesBlock `yaml:"resources"`
	Experimental *exportedExperimental  `yaml:"experimental,omitempty"`
}

type bundleBlock struct {
	Name string `yaml:"name"`
}

// exportedExperimental carries experimental.immutable_folder. air run uploads the
// synced code as a single content-addressed snapshot (/api/2.0/repos/snapshots)
// rather than per-file — the mechanism that mirrors AIR's own zip-and-fingerprint
// model. Requires the direct deployment engine, which the run path uses.
type exportedExperimental struct {
	ImmutableFolder bool `yaml:"immutable_folder"`
}

type exportedResourcesBlock struct {
	Jobs map[string]exportedJob `yaml:"jobs"`
}

type exportedJob struct {
	Name         string                `yaml:"name"`
	Tasks        []exportedTask        `yaml:"tasks"`
	Environments []exportedEnvironment `yaml:"environments"`
	// EnvironmentVariables holds env-var profiles, referenced by a task's
	// EnvironmentVariablesKey. Emitted only when the run declares env_variables or
	// secrets.
	EnvironmentVariables []exportedEnvVarProfile `yaml:"environment_variables,omitempty"`
	// Permissions are the job's ACL grants, from the run's permissions block.
	// Emitted only when the run declares any.
	Permissions []exportedPermission `yaml:"permissions,omitempty"`
}

// exportedPermission is one job ACL grant: a level plus exactly one principal.
// Matches the DABs job permissions shape.
type exportedPermission struct {
	Level                string `yaml:"level"`
	UserName             string `yaml:"user_name,omitempty"`
	GroupName            string `yaml:"group_name,omitempty"`
	ServicePrincipalName string `yaml:"service_principal_name,omitempty"`
}

type exportedTask struct {
	TaskKey        string `yaml:"task_key"`
	EnvironmentKey string `yaml:"environment_key"`
	// EnvironmentVariablesKey references a job-level environment_variables profile.
	// Omitted when the run has no env vars or secrets.
	EnvironmentVariablesKey string                `yaml:"environment_variables_key,omitempty"`
	MaxRetries              int                   `yaml:"max_retries"`
	TimeoutSeconds          int                   `yaml:"timeout_seconds,omitempty"`
	AiRuntimeTask           exportedAiRuntimeTask `yaml:"ai_runtime_task"`
}

// exportedEnvVarProfile is one entry in the job-level environment_variables list.
// Variables holds plain values inline and secrets as {{secrets/scope/key}}
// references, resolved by Jobs at run time.
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
// databricks-ai managed base environment rather than a bare channel.
const databricksAITokenPrefix = "databricks_ai_v"

// databricksAIBaseEnvironment is the system base_environment id the databricks-ai
// token resolves to.
const databricksAIBaseEnvironment = "workspace-base-environments/"

// convertToBundle maps a convertible runConfig to the emitted bundle, assuming
// checkBundleConvertible has already passed. command_path is emitted as a path
// relative to the bundle root; the bundle's translate_paths mutator rewrites it to
// the deployed location (for immutable_folder, under the content-addressed
// snapshot). Emitting ${workspace.file_path} directly would instead be validated as
// a local file and fail.
func convertToBundle(cfg *runConfig) *exportedBundle {
	task := exportedAiRuntimeTask{
		Experiment: cfg.ExperimentName,
		Deployments: []exportedDeployment{{
			// The deployment name is cosmetic (the wire payload carries none).
			Name:        "worker",
			CommandPath: bundleCommandScript,
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

	// One job-level env-var profile, referenced by the task's
	// environment_variables_key. Emitted only when there are vars or secrets.
	profiles := envVarProfiles(cfg)
	if len(profiles) > 0 {
		tsk.EnvironmentVariablesKey = aiRuntimeEnvVarsKey
	}

	// The experiment name is safe as a resource key: validateExperimentName already
	// guarantees the task_key charset.
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
					Permissions:          exportedPermissions(cfg),
				},
			},
		},
		Experimental: &exportedExperimental{ImmutableFolder: true},
	}
}

// exportedPermissions maps the run's permissions block to job ACL grants, or nil
// when none are declared. Each entry carries the level and its single principal;
// validation (exactly one principal, non-empty level) already ran in runConfig.
func exportedPermissions(cfg *runConfig) []exportedPermission {
	if len(cfg.Permissions) == 0 {
		return nil
	}
	out := make([]exportedPermission, 0, len(cfg.Permissions))
	for _, p := range cfg.Permissions {
		e := exportedPermission{Level: p.Level}
		switch {
		case p.UserName != nil:
			e.UserName = *p.UserName
		case p.GroupName != nil:
			e.GroupName = *p.GroupName
		case p.ServicePrincipalName != nil:
			e.ServicePrincipalName = *p.ServicePrincipalName
		}
		out = append(out, e)
	}
	return out
}

// envVarProfiles builds the job-level env-var profile list from the run's
// env_variables and secrets, or nil when there are none. Plain values are inline;
// each secret (ENV_VAR -> "scope/key") becomes a {{secrets/scope/key}} reference
// Jobs resolves at run time.
func envVarProfiles(cfg *runConfig) []exportedEnvVarProfile {
	if len(cfg.EnvVariables) == 0 && len(cfg.Secrets) == 0 {
		return nil
	}
	variables := make(map[string]string, len(cfg.EnvVariables)+len(cfg.Secrets))
	maps.Copy(variables, cfg.EnvVariables)
	for envVar, secretRef := range cfg.Secrets {
		variables[envVar] = "{{secrets/" + secretRef + "}}"
	}
	return []exportedEnvVarProfile{{
		EnvironmentVariablesKey: aiRuntimeEnvVarsKey,
		Variables:               variables,
	}}
}

// exportBundleEnvSpec resolves the serverless runtime selection: a
// "databricks_ai_v<N>" token selects the managed databricks-ai base_environment
// (torch + ML venv), while a bare numeric channel ("4", "5", ...) uses
// environment_version. It does not read process env, so the generated bundle is
// reproducible.
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
