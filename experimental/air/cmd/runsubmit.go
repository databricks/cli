package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/google/uuid"
)

// jobsRunsSubmitPath is the Jobs one-time-run endpoint. air builds the full
// payload and POSTs it here directly — the native ai_runtime_task is not modeled
// by the typed SDK, and we want no genai-mapi forwarding.
const jobsRunsSubmitPath = "/api/2.2/jobs/runs/submit"

// dlRuntimeImageEnv overrides the default deep-learning runtime image.
const dlRuntimeImageEnv = "DATABRICKS_DL_RUNTIME_IMAGE"

const defaultDlRuntimeImage = "CLIENT-GPU-4"

// aiRuntimeEnvironmentKey ties the task to the serverless environment that
// carries the runtime channel.
const aiRuntimeEnvironmentKey = "default"

// aiRuntimeCompute is a deployment's accelerator request.
type aiRuntimeCompute struct {
	AcceleratorType  string `json:"accelerator_type"`
	AcceleratorCount int    `json:"accelerator_count"`
}

// aiRuntimeDeployment is one worker deployment of the run.
type aiRuntimeDeployment struct {
	CommandPath string           `json:"command_path"`
	Compute     aiRuntimeCompute `json:"compute"`
}

// aiRuntimeTask is the native AI Runtime task. It routes straight to the training
// service — no genai-mapi forwarding. The proto is lean: env vars, secrets,
// requirements, and hyperparameters are staged as workspace files co-located with
// command.sh (see runupload.go), not carried inline.
type aiRuntimeTask struct {
	Experiment                string                `json:"experiment"`
	Deployments               []aiRuntimeDeployment `json:"deployments"`
	MlflowRun                 string                `json:"mlflow_run,omitempty"`
	MlflowExperimentDirectory string                `json:"mlflow_experiment_directory,omitempty"`
}

// environmentSpec carries the bare runtime channel ("4", "5", ...).
type environmentSpec struct {
	EnvironmentVersion string `json:"environment_version"`
}

// jobEnvironment is the serverless environment a task references for its runtime.
type jobEnvironment struct {
	EnvironmentKey string          `json:"environment_key"`
	Spec           environmentSpec `json:"spec"`
}

// submitTask is the single task air submits: a native ai_runtime_task.
type submitTask struct {
	TaskKey        string        `json:"task_key"`
	RunIf          string        `json:"run_if"`
	AiRuntimeTask  aiRuntimeTask `json:"ai_runtime_task"`
	EnvironmentKey string        `json:"environment_key"`
	MaxRetries     int           `json:"max_retries"`
	RetryOnTimeout bool          `json:"retry_on_timeout,omitempty"`
}

// jobsSubmitRun is the Jobs runs/submit payload.
type jobsSubmitRun struct {
	RunName          string           `json:"run_name"`
	TimeoutSeconds   int              `json:"timeout_seconds,omitempty"`
	Tasks            []submitTask     `json:"tasks"`
	Environments     []jobEnvironment `json:"environments"`
	BudgetPolicyID   string           `json:"budget_policy_id,omitempty"`
	IdempotencyToken string           `json:"idempotency_token,omitempty"`
}

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
func buildSubmitPayload(cfg *runConfig, commandPath, dlImage string) jobsSubmitRun {
	task := aiRuntimeTask{
		Experiment: cfg.ExperimentName,
		Deployments: []aiRuntimeDeployment{{
			CommandPath: commandPath,
			Compute: aiRuntimeCompute{
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

	st := submitTask{
		TaskKey:        cfg.ExperimentName,
		RunIf:          "ALL_SUCCESS",
		AiRuntimeTask:  task,
		EnvironmentKey: aiRuntimeEnvironmentKey,
		MaxRetries:     cfg.maxRetries(),
	}
	// max_retries 0 (no retries) is sent explicitly; retry_on_timeout only
	// applies when retries are allowed.
	st.RetryOnTimeout = st.MaxRetries > 0

	return jobsSubmitRun{
		RunName:        cfg.ExperimentName,
		TimeoutSeconds: cfg.timeoutSeconds(),
		Tasks:          []submitTask{st},
		Environments: []jobEnvironment{{
			EnvironmentKey: aiRuntimeEnvironmentKey,
			Spec:           environmentSpec{EnvironmentVersion: dlImage},
		}},
	}
}

// jobsSubmitClient submits one-time runs through the Jobs API.
type jobsSubmitClient struct {
	c *client.DatabricksClient
}

func newJobsSubmitClient(w *databricks.WorkspaceClient) (*jobsSubmitClient, error) {
	c, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &jobsSubmitClient{c: c}, nil
}

type submitRunResponse struct {
	RunID int64 `json:"run_id,omitempty"`
}

// submit POSTs the payload to runs/submit and returns the new run_id.
func (j *jobsSubmitClient) submit(ctx context.Context, payload jobsSubmitRun) (int64, error) {
	var resp submitRunResponse
	if err := j.c.Do(ctx, http.MethodPost, jobsRunsSubmitPath, auth.WorkspaceIDHeaders(j.c.Config), nil, payload, &resp); err != nil {
		return 0, err
	}
	return resp.RunID, nil
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
	// Resolving usage_policy_name to a budget policy id and packaging a
	// code_source snapshot are not ported yet; reject rather than silently drop.
	if cfg.UsagePolicyName != nil {
		return 0, "", errors.New("usage_policy_name is not yet supported")
	}
	if cfg.CodeSource != nil {
		return 0, "", errors.New("code_source is not yet supported")
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

	runtimeVersion, _ := cfg.runtimeVersion()
	payload := buildSubmitPayload(cfg, path.Join(funcDir, commandScriptName), dlRuntimeImage(ctx, runtimeVersion))
	payload.IdempotencyToken = token

	jc, err := newJobsSubmitClient(w)
	if err != nil {
		return 0, "", err
	}
	runID, err := jc.submit(ctx, payload)
	if err != nil {
		return 0, "", err
	}

	dashboardURL := strings.TrimRight(w.Config.Host, "/") + "/jobs/runs/" + strconv.FormatInt(runID, 10)
	return runID, dashboardURL, nil
}
