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
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// dlRuntimeImageEnv overrides the default deep-learning runtime image.
const dlRuntimeImageEnv = "DATABRICKS_DL_RUNTIME_IMAGE"

const defaultDlRuntimeImage = "CLIENT-GPU-6"

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

func buildRunEnvironment(dlImage string, deps []string) *compute.Environment {
	environment := &compute.Environment{}
	if strings.HasPrefix(dlImage, databricksAIPrefix) {
		environment.BaseEnvironment = "workspace-base-environments/" + dlImage
	} else {
		environment.EnvironmentVersion = dlImage
	}
	if len(deps) > 0 {
		environment.Dependencies = deps
	}
	return environment
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
	deployment := jobs.DeploymentSpec{
		Compute: jobs.ComputeSpec{
			AcceleratorType:  jobs.ComputeSpecAcceleratorType(cfg.Compute.AcceleratorType),
			AcceleratorCount: cfg.Compute.NumAccelerators,
		},
	}
	if len(cfg.Containers) == 0 {
		deployment.CommandPath = commandPath
	}
	task := jobs.AiRuntimeTask{
		Experiment:     cfg.ExperimentName,
		Deployments:    []jobs.DeploymentSpec{deployment},
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
	envSpec := buildRunEnvironment(dlImage, deps)

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

type submittedContainer struct {
	Name                  string
	CommandPath           string
	Ranks                 []int
	UnityCatalogImagePath string
}

func submitRun(ctx context.Context, w *databricks.WorkspaceClient, payload jobs.SubmitRun, poolID, priorityClass, unityCatalogImagePath string, containers []submittedContainer) (int64, error) {
	// None of these fields are modeled by the SDK's AiRuntimeTask, so a run that
	// sets any of them has to go through the raw /api/2.2 body. priority_class only
	// ever appears alongside a pool (validation enforces it), but route on
	// all of them so none can be silently dropped.
	if poolID == "" && priorityClass == "" && unityCatalogImagePath == "" && len(containers) == 0 {
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
	if err := injectPoolFields(body, poolID, priorityClass); err != nil {
		return 0, err
	}
	if unityCatalogImagePath != "" {
		aiRuntimeTask, err := aiRuntimeTaskFromSubmitBody(body)
		if err != nil {
			return 0, err
		}
		aiRuntimeTask["unity_catalog_image_path"] = unityCatalogImagePath
	}
	if len(containers) > 0 {
		if err := injectContainers(body, containers); err != nil {
			return 0, err
		}
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

func injectContainers(body map[string]any, containers []submittedContainer) error {
	aiRuntimeTask, err := aiRuntimeTaskFromSubmitBody(body)
	if err != nil {
		return err
	}
	deployments, ok := aiRuntimeTask["deployments"].([]any)
	if !ok || len(deployments) != 1 {
		return errors.New("AIR submit payload must contain exactly one deployment")
	}
	deployment, ok := deployments[0].(map[string]any)
	if !ok {
		return errors.New("AIR submit payload deployment has an invalid shape")
	}
	// The SDK's DeploymentSpec marshals its required command_path even when
	// empty. Container runs have their own commands, so omit that wire field.
	delete(deployment, "command_path")
	raw := make([]any, 0, len(containers))
	for _, container := range containers {
		ranks := make([]any, len(container.Ranks))
		for i, rank := range container.Ranks {
			ranks[i] = rank
		}
		raw = append(raw, map[string]any{
			"name":                     container.Name,
			"command_path":             container.CommandPath,
			"ranks":                    ranks,
			"unity_catalog_image_path": container.UnityCatalogImagePath,
		})
	}
	deployment["containers"] = raw
	return nil
}

// injectPoolFields sets the pool-only fields the SDK does not model onto the
// decoded submit body: priority_class rides directly on the ai_runtime_task,
// while provisioned_capacity_id (the wire name for the pool) rides on the
// deployment's compute spec. Each is set only when non-empty.
func injectPoolFields(body map[string]any, poolID, priorityClass string) error {
	aiRuntimeTask, err := aiRuntimeTaskFromSubmitBody(body)
	if err != nil {
		return err
	}
	if priorityClass != "" {
		aiRuntimeTask["priority_class"] = priorityClass
	}
	if poolID != "" {
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
		computeSpec["provisioned_capacity_id"] = poolID
	}
	return nil
}

// aiRuntimeTaskFromSubmitBody navigates a decoded runs/submit body to its single
// ai_runtime_task map, erroring if the payload isn't the expected single-task shape.
func aiRuntimeTaskFromSubmitBody(body map[string]any) (map[string]any, error) {
	tasks, ok := body["tasks"].([]any)
	if !ok || len(tasks) != 1 {
		return nil, errors.New("AIR submit payload must contain exactly one task")
	}
	task, ok := tasks[0].(map[string]any)
	if !ok {
		return nil, errors.New("AIR submit payload task has an invalid shape")
	}
	aiRuntimeTask, ok := task["ai_runtime_task"].(map[string]any)
	if !ok {
		return nil, errors.New("AIR submit payload is missing ai_runtime_task")
	}
	return aiRuntimeTask, nil
}

// effectiveIdempotencyToken resolves the token sent to validation and submit:
// the --idempotency-key flag wins, then the config's token, else a generated one.
func effectiveIdempotencyToken(flag string, cfg *runConfig) string {
	token := flag
	if token == "" && cfg.IdempotencyToken != nil {
		token = *cfg.IdempotencyToken
	}
	if token == "" {
		token = uuid.NewString()
	}
	return token
}

// validateIdempotencyToken keeps the CLI's submission guard as a fallback when
// backend validation is unavailable or does not yet enforce the same rule.
func validateIdempotencyToken(token string) error {
	if len(token) > 64 {
		return fmt.Errorf("idempotency token must be 64 characters or less, got %d", len(token))
	}
	return nil
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

// stageRunArtifacts uploads the launch files and optional code snapshot in
// parallel. It returns only after both branches complete, so callers can safely
// build and submit a payload that references their remote paths. Snapshot
// sidecar writes must keep CreateParentDirectories because launch-directory
// creation is concurrent, not ordered before snapshot staging.
func stageRunArtifacts(ctx context.Context, launchWriter fileWriter, items []uploadItem, stageSnapshot func(context.Context) (snapshotResult, error)) (snapshotResult, error) {
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return uploadArtifacts(groupCtx, launchWriter, items)
	})

	var snap snapshotResult
	if stageSnapshot != nil {
		group.Go(func() error {
			var err error
			snap, err = stageSnapshot(groupCtx)
			return err
		})
	}

	err := group.Wait()
	// Preserve measurements from attempted snapshot phases even if staging fails.
	return snap, err
}

// submitWorkload runs the submit happy path: ensure the experiment directory,
// upload the launch artifacts, assemble the Jobs payload, and submit it. It
// returns the new run_id and its dashboard URL. showProgress enables the stderr
// staging spinner (text mode only).
func submitWorkload(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, configPath, idempotencyKey string, showProgress bool) (runID int64, dashboardURL string, err error) {
	start := time.Now()
	var snap snapshotResult
	defer func() { logRunEvent(ctx, cfg, snap, runID, time.Since(start), err) }()

	// Compute and validate the actual submission path before creating artifacts.
	base, funcDir, commandPath, err := prospectiveLaunchPaths(ctx, w, cfg)
	if err != nil {
		return 0, "", err
	}
	containers := submittedContainers(cfg, funcDir)
	token := effectiveIdempotencyToken(idempotencyKey, cfg)

	// Validate server-side before uploading, so invalid configs leave no artifacts.
	if err := preflightValidate(ctx, w, cfg, commandPath, containers, token); err != nil {
		return 0, "", err
	}

	// Retain the local guard after backend validation so an unavailable endpoint
	// cannot let a bad key reach upload or submission.
	if err := validateIdempotencyToken(token); err != nil {
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

	fc, err := filer.NewWorkspaceFilesClient(w, funcDir)
	if err != nil {
		return 0, "", err
	}
	items, err := buildArtifacts(cfg)
	if err != nil {
		return 0, "", err
	}

	// Package and upload the code snapshot, if any; the remote code_source_path
	// rides the ai_runtime_task. A run with no code_source leaves it empty.
	// Snapshot is the only code_source type.
	var stageSnapshot func(context.Context) (snapshotResult, error)
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil {
		// Default snapshot tarballs land in the user's shared repo_snapshots dir;
		// uploadSnapshot replaces this when remote_volume is configured.
		snapshotArtifactPath := path.Join(base, ".air", "repo_snapshots")
		stageSnapshot = func(ctx context.Context) (snapshotResult, error) {
			// Sidecars land in the run's launch dir (funcDir) via fc, next to command.sh.
			return uploadSnapshot(ctx, w, cfg.CodeSource.Snapshot, configPath, snapshotArtifactPath, fc, funcDir)
		}
	}

	err = withSpinner(ctx, showProgress, "Staging run artifacts…", func() error {
		var stageErr error
		snap, stageErr = stageRunArtifacts(ctx, fc, items, stageSnapshot)
		return stageErr
	})
	if err != nil {
		return 0, "", err
	}

	runtimeVersion, _ := cfg.runtimeVersion()
	payload := buildSubmitPayload(cfg, commandPath, dlRuntimeImage(ctx, runtimeVersion), usagePolicyID, snap, deps)
	payload.IdempotencyToken = token

	// The pool id is sent on the wire as provisioned_capacity_id (the backend's
	// name for a GPU pool); only the user-facing YAML field is pool_id.
	poolID := ""
	if cfg.Compute.PoolID != nil {
		poolID = *cfg.Compute.PoolID
	}
	priorityClass := ""
	if cfg.Compute.PriorityClass != nil {
		priorityClass = *cfg.Compute.PriorityClass
	}
	// Submit returns as soon as the run is created; we don't wait for it to finish.
	// Permissions are granted by the caller, after the submit result is shown, so
	// the best-effort grant never delays the success line.
	runID, err = submitRun(ctx, w, payload, poolID, priorityClass, cfg.unityCatalogImagePath(), containers)
	if err != nil {
		return 0, "", err
	}

	dashboardURL = strings.TrimRight(w.Config.Host, "/") + "/jobs/runs/" + strconv.FormatInt(runID, 10)
	return runID, dashboardURL, nil
}

func submittedContainers(cfg *runConfig, funcDir string) []submittedContainer {
	containers := make([]submittedContainer, 0, len(cfg.Containers))
	for _, container := range cfg.Containers {
		containers = append(containers, submittedContainer{
			Name:                  container.Name,
			CommandPath:           path.Join(funcDir, "containers", container.Name, commandScriptName),
			Ranks:                 container.Ranks,
			UnityCatalogImagePath: container.UnityCatalogImage,
		})
	}
	return containers
}
