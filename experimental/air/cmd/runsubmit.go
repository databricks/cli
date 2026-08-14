package aircmd

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
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
func buildSubmitPayload(cfg *runConfig, commandPath, dlImage, usagePolicyID string, snap snapshotResult, deps []string) jobs.SubmitRun {
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
	}
	if cfg.MLflowRunName != nil {
		task.MlflowRun = *cfg.MLflowRunName
	}
	if cfg.MLflowExperimentDirectory != nil {
		task.MlflowExperimentDirectory = *cfg.MLflowExperimentDirectory
	}

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

	// Carry the user's declared deps inline on spec.dependencies; the AI Runtime
	// backend installs them via --deps-config. The SDK marshaler drops nil and empty
	// slices, so a no-deps run omits the key.
	envSpec := &compute.Environment{EnvironmentVersion: dlImage}
	if len(deps) > 0 {
		envSpec.Dependencies = deps
	}

	return jobs.SubmitRun{
		RunName: cfg.ExperimentName,
		// budget_policy_id matches what the Python CLI and `ssh connect` send;
		// usage_policy_id is the newer alias for the same thing on SubmitRun.
		BudgetPolicyId: usagePolicyID,
		TimeoutSeconds: cfg.timeoutSeconds(),
		Tasks:          []jobs.SubmitTask{st},
		Environments: []jobs.JobEnvironment{{
			EnvironmentKey: aiRuntimeEnvironmentKey,
			Spec:           envSpec,
		}},
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

// withSpinner runs fn, showing an stderr spinner labeled msg when show is true.
// The spinner auto-degrades to nothing on a non-interactive terminal; show is
// false in JSON mode so the stdout envelope stream stays clean.
func withSpinner(ctx context.Context, show bool, msg string, fn func() error) error {
	if !show {
		return fn()
	}
	sp := cmdio.NewSpinner(ctx)
	sp.Update(msg)
	defer sp.Close()
	return fn()
}

// submitWorkload runs the submit happy path: ensure the experiment directory,
// upload the launch artifacts, assemble the Jobs payload, and submit it. It
// returns the new run_id and its dashboard URL. showProgress enables the
// stderr upload/packaging spinners (text mode only).
func submitWorkload(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, configPath, idempotencyKey string, showProgress bool) (int64, string, error) {
	// Compute the launch dir and command_path up front — a read-only workspace lookup plus a
	// local path build, no writes yet — so the pre-flight validates the real command_path. The
	// same path is reused for the upload and submit below, so the validated path is the submitted
	// one.
	base, err := userWorkspaceDir(ctx, w)
	if err != nil {
		return 0, "", err
	}
	runName := ""
	if cfg.MLflowRunName != nil {
		runName = *cfg.MLflowRunName
	}
	funcDir := cliLaunchDir(base, cfg.ExperimentName, runName)
	commandPath := path.Join(funcDir, commandScriptName)

	// Pre-flight the config server-side before any upload, so a bad config fails with the
	// backend's field-level errors and no orphaned artifacts.
	if err := preflightValidate(ctx, w, cfg, commandPath); err != nil {
		return 0, "", err
	}

	// Resolve the idempotency token first so a bad key fails before any upload,
	// and before the policy lookup below spends a round trip on it.
	token, err := submitToken(idempotencyKey, cfg)
	if err != nil {
		return 0, "", err
	}

	// Resolve the usage policy to its id next, so a bad name fails fast with a
	// clear (caller-fixable) message before we upload any artifacts. Validation
	// guarantees name and id are mutually exclusive: a literal id is used as-is, a
	// name is resolved against the workspace.
	usagePolicyID := ""
	if cfg.UsagePolicyID != nil {
		usagePolicyID = strings.TrimSpace(*cfg.UsagePolicyID)
	}
	if cfg.UsagePolicyName != nil {
		usagePolicyID, err = resolveUsagePolicyIDByName(ctx, w, *cfg.UsagePolicyName)
		if err != nil {
			return 0, "", err
		}
	}

	// Resolve dependencies before any upload too, so a bad requirements file fails
	// fast without leaving orphaned artifacts in the workspace.
	deps, fileVersion, err := environmentDependencies(cfg, configPath)
	if err != nil {
		return 0, "", err
	}

	experimentDir := ""
	if cfg.MLflowExperimentDirectory != nil {
		experimentDir = *cfg.MLflowExperimentDirectory
	}
	if err := ensureExperimentDirectory(ctx, w, experimentDir); err != nil {
		return 0, "", err
	}

	fc, err := filer.NewWorkspaceFilesClient(w, funcDir)
	if err != nil {
		return 0, "", err
	}
	items, err := buildArtifacts(cfg, configPath)
	if err != nil {
		return 0, "", err
	}
	if err := withSpinner(ctx, showProgress, "Uploading yaml configuration files…", func() error {
		return uploadArtifacts(ctx, fc, items)
	}); err != nil {
		return 0, "", err
	}

	// Package and upload the code snapshot, if any, via DABs' artifact-upload
	// plumbing; the remote code_source_path rides the ai_runtime_task. A run with no
	// code_source leaves it empty. Snapshot is the only code_source type.
	var snap snapshotResult
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil {
		// Sidecars land in the run's launch dir (funcDir) via fc, next to command.sh.
		err = withSpinner(ctx, showProgress, "Packaging code snapshot…", func() error {
			var e error
			snap, e = snapshotViaDABsUpload(ctx, w, cfg.CodeSource.Snapshot, configPath, fc, funcDir)
			return e
		})
		if err != nil {
			return 0, "", err
		}
	}

	// Top-level environment.version wins; for file-form deps it is disallowed, so
	// fall back to the version declared inside the requirements file.
	runtimeVersion, ok := cfg.runtimeVersion()
	if !ok {
		runtimeVersion = fileVersion
	}
	payload := buildSubmitPayload(cfg, commandPath, dlRuntimeImage(ctx, runtimeVersion), usagePolicyID, snap, deps)
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
