package aircmd

import (
	"context"
	"errors"
	"fmt"
	"path"
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

// buildSubmitPayload assembles the runs/submit payload. commandPath is the
// workspace path of the uploaded command.sh; dlImage is the runtime channel.
//
// max_retries is always sent (including 0) so the user's YAML value is honored:
// setting it to 0 explicitly disables retries rather than falling back to the
// server default. retry_on_timeout is sent only when retries are allowed, and is
// omitempty so the wire form matches the Python CLI (which never emits a bare
// "false"). Jobs performs the retries — each attempt is a fresh AI Runtime
// workload.
func buildSubmitPayload(cfg *runConfig, commandPath, dlImage string, snap snapshotResult) jobs.SubmitRun {
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
		// TEMP: git_state_path / git_diff_path are intentionally NOT sent. The typed
		// jobs.AiRuntimeTask (and its source proto, ai_runtime_task.proto) has no such
		// fields, so the typed SDK path cannot carry them. This is safe today because
		// nothing in the backend consumes those fields — the AI Runtime task proto
		// never declared them, so even the Python CLI's raw-JSON values were dropped
		// on deserialization. The git_state.json / git_diff.patch sidecars are still
		// uploaded next to the tarball (see snapshot.go) for human inspection.
		// If the backend later adds these fields to the proto, regenerate the SDK and
		// wire snap.GitStatePath / snap.GitDiffPath back in here.
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

	return jobs.SubmitRun{
		RunName:        cfg.ExperimentName,
		TimeoutSeconds: cfg.timeoutSeconds(),
		Tasks:          []jobs.SubmitTask{st},
		Environments: []jobs.JobEnvironment{{
			EnvironmentKey: aiRuntimeEnvironmentKey,
			Spec:           &compute.Environment{EnvironmentVersion: dlImage},
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

// submitWorkload runs the submit happy path: ensure the experiment directory,
// upload the launch artifacts, assemble the Jobs payload, and submit it. It
// returns the new run_id and its dashboard URL.
func submitWorkload(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, configPath, idempotencyKey string) (int64, string, error) {
	// Resolving usage_policy_name to a budget policy id is not ported yet; reject
	// rather than silently drop.
	if cfg.UsagePolicyName != nil {
		return 0, "", errors.New("usage_policy_name is not yet supported")
	}

	// Resolve the idempotency token first so a bad key fails before any upload.
	token, err := submitToken(idempotencyKey, cfg)
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

	base, err := userWorkspaceDir(ctx, w)
	if err != nil {
		return 0, "", err
	}
	runName := ""
	if cfg.MLflowRunName != nil {
		runName = *cfg.MLflowRunName
	}
	funcDir := cliLaunchDir(base, cfg.ExperimentName, runName)

	fc, err := filer.NewWorkspaceFilesClient(w, funcDir)
	if err != nil {
		return 0, "", err
	}
	items, err := buildArtifacts(cfg, configPath)
	if err != nil {
		return 0, "", err
	}
	if err := uploadArtifacts(ctx, fc, items); err != nil {
		return 0, "", err
	}

	// Package and upload the code snapshot, if any. The resulting paths ride on the
	// ai_runtime_task; a run with no code_source leaves them empty. Snapshot is the
	// only code_source type; guard against a nil block so snapshotCodeSource never
	// dereferences a missing snapshot.
	var snap snapshotResult
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil {
		snap, err = snapshotCodeSource(ctx, w, cfg.CodeSource.Snapshot, configPath, base, funcDir)
		if err != nil {
			return 0, "", err
		}
	}

	runtimeVersion, _ := cfg.runtimeVersion()
	payload := buildSubmitPayload(cfg, path.Join(funcDir, commandScriptName), dlRuntimeImage(ctx, runtimeVersion), snap)
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
