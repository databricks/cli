package aircmd

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
)

// dlRuntimeImageEnv overrides the default deep-learning runtime image.
const dlRuntimeImageEnv = "DATABRICKS_DL_RUNTIME_IMAGE"

const defaultDlRuntimeImage = "CLIENT-GPU-4"

// aiRuntimeEnvironmentKey ties the task to the serverless environment that
// carries the runtime channel.
const aiRuntimeEnvironmentKey = "default"

// dlRuntimeImage resolves the bare runtime channel (config version, else env,
// else default), always stripping the CLIENT-GPU- prefix.
func dlRuntimeImage(ctx context.Context, runtimeVersion string) string {
	img := runtimeVersion
	if img == "" {
		img = env.Get(ctx, dlRuntimeImageEnv)
	}
	if img == "" {
		img = defaultDlRuntimeImage
	}
	return strings.TrimPrefix(img, "CLIENT-GPU-")
}

// environmentDependencies resolves the user's declared dependencies as a flat
// list to carry inline on the serverless environment's spec.dependencies: the
// inline list directly, or the dependencies read from a requirements file
// (resolved against the config's directory). For file-form deps it also returns
// the version declared inside that file, which selects the runtime image since
// top-level environment.version is not allowed there. Returns nil when none are
// declared.
func environmentDependencies(cfg *runConfig, configPath string) (deps []string, fileVersion string, err error) {
	if deps, ok := cfg.inlineDependencies(); ok {
		return deps, "", nil
	}
	if reqPath, ok := cfg.requirementsFile(); ok {
		if !filepath.IsAbs(reqPath) {
			reqPath = filepath.Join(filepath.Dir(configPath), reqPath)
		}
		return readRequirementsDependencies(reqPath)
	}
	return nil, "", nil
}

// buildSubmitPayload assembles the runs/submit payload. commandPath is the
// workspace path of the uploaded command.sh; dlImage is the runtime channel;
// usagePolicyID is the already-resolved policy id ("" when the run has none);
// deps is the user's declared dependencies (nil when none are declared).
//
// max_retries is always sent (including 0) so the user's YAML value is honored:
// setting it to 0 explicitly disables retries rather than falling back to the
// server default. retry_on_timeout is sent only when retries are allowed, and is
// omitempty so the wire form matches the Python CLI (which never emits a bare
// "false"). Jobs performs the retries — each attempt is a fresh AI Runtime
// workload.
// buildAiRuntimeTask assembles the ai_runtime_task shared by the ephemeral
// (SubmitTask) and persistent (Task) payloads.
func buildAiRuntimeTask(cfg *runConfig, commandPath string, snap snapshotResult) jobs.AiRuntimeTask {
	task := jobs.AiRuntimeTask{
		Experiment: cfg.ExperimentName,
		Deployments: []jobs.DeploymentSpec{{
			CommandPath: commandPath,
			Compute: jobs.ComputeSpec{
				AcceleratorType:  jobs.ComputeSpecAcceleratorType(cfg.Compute.AcceleratorType),
				AcceleratorCount: cfg.Compute.NumAccelerators,
			},
		}},
		CodeSourcePath: snap.CodeSourcePath,
		// NOTE: docker_image_url is intentionally not set here yet. The field was
		// added to jobs.AiRuntimeTask in databricks-sdk-go after v0.170.0, which the
		// CLI has not bumped to. prepareDockerImage already verifies the image is
		// registered; passing it on the task lands in the follow-up PR once the SDK
		// bump + codegen is in.
	}
	if cfg.MLflowRunName != nil {
		task.MlflowRun = *cfg.MLflowRunName
	}
	if cfg.MLflowExperimentDirectory != nil {
		task.MlflowExperimentDirectory = *cfg.MLflowExperimentDirectory
	}
	return task
}

// buildAiRuntimeEnvironments carries the user's declared deps inline on
// spec.dependencies; the AI Runtime backend installs them via --deps-config. The
// SDK marshaler drops nil and empty slices, so a no-deps run omits the key.
func buildAiRuntimeEnvironments(dlImage string, deps []string) []jobs.JobEnvironment {
	envSpec := &compute.Environment{EnvironmentVersion: dlImage}
	if len(deps) > 0 {
		envSpec.Dependencies = deps
	}
	return []jobs.JobEnvironment{{
		EnvironmentKey: aiRuntimeEnvironmentKey,
		Spec:           envSpec,
	}}
}

func buildSubmitPayload(cfg *runConfig, commandPath, dlImage, usagePolicyID string, snap snapshotResult, deps []string) jobs.SubmitRun {
	task := buildAiRuntimeTask(cfg, commandPath, snap)

	maxRetries := cfg.maxRetries()
	st := jobs.SubmitTask{
		TaskKey:        cfg.ExperimentName,
		RunIf:          jobs.RunIfAllSuccess,
		AiRuntimeTask:  &task,
		EnvironmentKey: aiRuntimeEnvironmentKey,
		MaxRetries:     maxRetries,
		// retry_on_timeout only makes sense when retries are allowed; otherwise
		// omit it (matches Python's native path, which sets retry_on_timeout only
		// under the same > 0 gate).
		RetryOnTimeout:  maxRetries > 0,
		ForceSendFields: []string{"MaxRetries"},
	}

	return jobs.SubmitRun{
		RunName: cfg.ExperimentName,
		// budget_policy_id matches what the Python CLI and `ssh connect` send;
		// usage_policy_id is the newer alias for the same thing on SubmitRun.
		BudgetPolicyId: usagePolicyID,
		TimeoutSeconds: cfg.timeoutSeconds(),
		Tasks:          []jobs.SubmitTask{st},
		Environments:   buildAiRuntimeEnvironments(dlImage, deps),
	}
}

// submitToken resolves the idempotency token: the --idempotency-key flag wins,
// then the config's token, else a generated one. Over-long tokens error rather
// than truncate, since truncation could make two distinct tokens collide.
func submitToken(flag string, cfg *runConfig) (string, error) {
	token := flag
	if token == "" && cfg.IdempotencyToken != nil {
		token = *cfg.IdempotencyToken
	}
	if token == "" {
		token = uuid.NewString()
	}
	if len(token) > 64 {
		return "", fmt.Errorf("idempotency token must be 64 characters or less, got %d", len(token))
	}
	return token, nil
}

// preparedWorkload holds everything resolved and uploaded for a workload, ready
// to become either an ephemeral SubmitRun (submitWorkload) or a persistent,
// scheduled CreateJob (createScheduledJob).
type preparedWorkload struct {
	commandPath   string
	dlImage       string
	usagePolicyID string
	snap          snapshotResult
	deps          []string
}

// prepareWorkload runs the shared pre-submit work: resolve the usage policy and
// dependencies, ensure the experiment directory, prepare any docker image, and
// upload the launch artifacts + code snapshot. Ordering fails cheap checks before
// any upload so a bad config leaves no orphaned artifacts in the workspace.
func prepareWorkload(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, configPath string) (preparedWorkload, error) {
	// Resolve the usage policy to its id first, so a bad name fails fast with a
	// clear (caller-fixable) message before we upload any artifacts. Validation
	// guarantees name and id are mutually exclusive: a literal id is used as-is, a
	// name is resolved against the workspace.
	usagePolicyID := ""
	if cfg.UsagePolicyID != nil {
		usagePolicyID = strings.TrimSpace(*cfg.UsagePolicyID)
	}
	if cfg.UsagePolicyName != nil {
		var err error
		usagePolicyID, err = resolveUsagePolicyIDByName(ctx, w, *cfg.UsagePolicyName)
		if err != nil {
			return preparedWorkload{}, err
		}
	}

	// Resolve dependencies before any upload too, so a bad requirements file fails
	// fast without leaving orphaned artifacts in the workspace.
	deps, fileVersion, err := environmentDependencies(cfg, configPath)
	if err != nil {
		return preparedWorkload{}, err
	}

	experimentDir := ""
	if cfg.MLflowExperimentDirectory != nil {
		experimentDir = *cfg.MLflowExperimentDirectory
	}
	if err := ensureExperimentDirectory(ctx, w, experimentDir); err != nil {
		return preparedWorkload{}, err
	}

	base, err := userWorkspaceDir(ctx, w)
	if err != nil {
		return preparedWorkload{}, err
	}

	// After the cheap workspace checks (a tag_policy=latest refresh can block for
	// minutes) but before any upload, so a bad image wastes no artifact work.
	if img := cfg.dockerImage(); img != nil {
		if err := prepareDockerImage(ctx, w, img); err != nil {
			return preparedWorkload{}, err
		}
	}

	runName := ""
	if cfg.MLflowRunName != nil {
		runName = *cfg.MLflowRunName
	}
	funcDir := cliLaunchDir(base, cfg.ExperimentName, runName)

	fc, err := filer.NewWorkspaceFilesClient(w, funcDir)
	if err != nil {
		return preparedWorkload{}, err
	}
	items, err := buildArtifacts(cfg, configPath)
	if err != nil {
		return preparedWorkload{}, err
	}
	if err := uploadArtifacts(ctx, fc, items); err != nil {
		return preparedWorkload{}, err
	}

	// Package and upload the code snapshot, if any, via DABs' artifact-upload
	// plumbing; the remote code_source_path rides the ai_runtime_task. A run with no
	// code_source leaves it empty. Snapshot is the only code_source type.
	var snap snapshotResult
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil {
		// Sidecars land in the run's launch dir (funcDir) via fc, next to command.sh.
		snap, err = snapshotViaDABsUpload(ctx, w, cfg.CodeSource.Snapshot, configPath, fc, funcDir)
		if err != nil {
			return preparedWorkload{}, err
		}
	}

	// Top-level environment.version wins; for file-form deps it is disallowed, so
	// fall back to the version declared inside the requirements file.
	runtimeVersion, ok := cfg.runtimeVersion()
	if !ok {
		runtimeVersion = fileVersion
	}

	return preparedWorkload{
		commandPath:   path.Join(funcDir, commandScriptName),
		dlImage:       dlRuntimeImage(ctx, runtimeVersion),
		usagePolicyID: usagePolicyID,
		snap:          snap,
		deps:          deps,
	}, nil
}

// submitWorkload runs the submit happy path: prepare the workload, assemble the
// ephemeral Jobs payload, and submit it. It returns the new run_id and its
// dashboard URL.
func submitWorkload(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, configPath, idempotencyKey string) (int64, string, error) {
	// Resolve the idempotency token first so a bad key fails before any upload.
	token, err := submitToken(idempotencyKey, cfg)
	if err != nil {
		return 0, "", err
	}

	prep, err := prepareWorkload(ctx, w, cfg, configPath)
	if err != nil {
		return 0, "", err
	}

	payload := buildSubmitPayload(cfg, prep.commandPath, prep.dlImage, prep.usagePolicyID, prep.snap, prep.deps)
	payload.IdempotencyToken = token

	// Submit returns as soon as the run is created; we don't wait for it to finish.
	wait, err := w.Jobs.Submit(ctx, payload)
	if err != nil {
		return 0, "", err
	}
	runID := wait.RunId

	dashboardURL := strings.TrimRight(w.Config.Host, "/") + "/jobs/runs/" + strconv.FormatInt(runID, 10)
	return runID, dashboardURL, nil
}
