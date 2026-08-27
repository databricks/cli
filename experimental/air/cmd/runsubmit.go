package aircmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
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
	if cfg.MLflowArtifactLocation != nil {
		task.MlflowArtifactLocation = *cfg.MLflowArtifactLocation
	}
	task.DockerImageUrl = cfg.dockerImageURL()

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

func submitRun(ctx context.Context, w *databricks.WorkspaceClient, payload jobs.SubmitRun, provisionedCapacityID string) (int64, error) {
	if provisionedCapacityID == "" {
		wait, err := w.Jobs.Submit(ctx, payload)
		if err != nil {
			return 0, err
		}
		return wait.RunId, nil
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal AIR submit payload: %w", err)
	}
	var body map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return 0, fmt.Errorf("failed to decode AIR submit payload: %w", err)
	}
	if err := injectProvisionedCapacityID(body, provisionedCapacityID); err != nil {
		return 0, err
	}

	apiClient, err := client.New(w.Config)
	if err != nil {
		return 0, fmt.Errorf("failed to create API client: %w", err)
	}
	var response jobs.SubmitRunResponse
	err = apiClient.Do(ctx, http.MethodPost, "/api/2.2/jobs/runs/submit", auth.WorkspaceIDHeaders(w.Config), nil, body, &response)
	if err != nil {
		return 0, err
	}
	return response.RunId, nil
}

func injectProvisionedCapacityID(body map[string]any, provisionedCapacityID string) error {
	tasks, ok := body["tasks"].([]any)
	if !ok || len(tasks) != 1 {
		return errors.New("AIR submit payload must contain exactly one task")
	}
	task, ok := tasks[0].(map[string]any)
	if !ok {
		return errors.New("AIR submit payload task has an invalid shape")
	}
	aiRuntimeTask, ok := task["ai_runtime_task"].(map[string]any)
	if !ok {
		return errors.New("AIR submit payload is missing ai_runtime_task")
	}
	deployments, ok := aiRuntimeTask["deployments"].([]any)
	if !ok || len(deployments) != 1 {
		return errors.New("AIR submit payload must contain exactly one deployment")
	}
	deployment, ok := deployments[0].(map[string]any)
	if !ok {
		return errors.New("AIR submit payload deployment has an invalid shape")
	}
	computeSpec, ok := deployment["compute"].(map[string]any)
	if !ok {
		return errors.New("AIR submit payload is missing deployment compute")
	}
	computeSpec["provisioned_capacity_id"] = provisionedCapacityID
	return nil
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

	deps, _ := cfg.inlineDependencies()

	experimentDir := ""
	if cfg.MLflowExperimentDirectory != nil {
		experimentDir = *cfg.MLflowExperimentDirectory
	}
	if err := ensureExperimentDirectory(ctx, w, experimentDir); err != nil {
		return 0, "", err
	}

	// After the cheap validations but before any upload: verify the custom image is
	// registered (and, under tag_policy=latest, re-resolve it — a refresh can block
	// for minutes), so a bad or unregistered image wastes no artifact work.
	if img := cfg.dockerImage(); img != nil {
		if err := prepareDockerImage(ctx, w, img); err != nil {
			return 0, "", err
		}
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

	runtimeVersion, _ := cfg.runtimeVersion()
	payload := buildSubmitPayload(cfg, commandPath, dlRuntimeImage(ctx, runtimeVersion), usagePolicyID, snap, deps)
	payload.IdempotencyToken = token

	provisionedCapacityID := ""
	if cfg.Compute.ProvisionedCapacityID != nil {
		provisionedCapacityID = *cfg.Compute.ProvisionedCapacityID
	}
	// Submit returns as soon as the run is created; we don't wait for it to finish.
	runID, err := submitRun(ctx, w, payload, provisionedCapacityID)
	if err != nil {
		return 0, "", err
	}

	dashboardURL := strings.TrimRight(w.Config.Host, "/") + "/jobs/runs/" + strconv.FormatInt(runID, 10)
	return runID, dashboardURL, nil
}
