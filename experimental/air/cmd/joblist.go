package aircmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

// jobsRunsListPath is the Jobs runs/list endpoint. We call it directly (rather
// than via the typed SDK) because the SDK's RunTask omits ai_runtime_task, the
// task type the AI runtime now submits.
const jobsRunsListPath = "/api/2.2/jobs/runs/list"

type jobsRunsListResponse struct {
	Runs          []jobRun `json:"runs"`
	NextPageToken string   `json:"next_page_token"`
}

type jobRun struct {
	RunID           int64     `json:"run_id"`
	RunName         string    `json:"run_name"`
	CreatorUserName string    `json:"creator_user_name"`
	StartTime       int64     `json:"start_time"`
	EndTime         int64     `json:"end_time"`
	State           jobState  `json:"state"`
	Tasks           []jobTask `json:"tasks"`
}

type jobState struct {
	LifeCycleState string `json:"life_cycle_state"`
	ResultState    string `json:"result_state"`
}

type jobTask struct {
	RunID            int64             `json:"run_id"`
	StartTime        int64             `json:"start_time"`
	EndTime          int64             `json:"end_time"`
	AiRuntimeTask    *aiRuntimeTask    `json:"ai_runtime_task"`
	GenAiComputeTask *genAiComputeTask `json:"gen_ai_compute_task"`
	ForEachTask      *forEachTask      `json:"for_each_task"`
}

type forEachTask struct {
	Task jobTask `json:"task"`
}

// aiRuntimeTask is the current AI runtime task shape; deployments[0].compute
// carries the accelerator info.
type aiRuntimeTask struct {
	Experiment  string            `json:"experiment"`
	Deployments []aiRuntimeDeploy `json:"deployments"`
}

type aiRuntimeDeploy struct {
	Compute airCompute `json:"compute"`
}

type airCompute struct {
	AcceleratorType  string `json:"accelerator_type"`
	AcceleratorCount int    `json:"accelerator_count"`
}

// genAiComputeTask is the legacy task shape, still recognized for older runs.
type genAiComputeTask struct {
	TrainingScriptPath   string        `json:"training_script_path"`
	MlflowExperimentName string        `json:"mlflow_experiment_name"`
	Compute              *genAiCompute `json:"compute"`
}

type genAiCompute struct {
	GpuType string `json:"gpu_type"`
	NumGpus int    `json:"num_gpus"`
}

// airTask returns the run's AI runtime / legacy GenAI task, unwrapping a foreach
// sweep when present.
func (r *jobRun) airTask() (*aiRuntimeTask, *genAiComputeTask) {
	if len(r.Tasks) == 0 {
		return nil, nil
	}
	t := r.Tasks[0]
	if t.AiRuntimeTask != nil || t.GenAiComputeTask != nil {
		return t.AiRuntimeTask, t.GenAiComputeTask
	}
	if t.ForEachTask != nil {
		return t.ForEachTask.Task.AiRuntimeTask, t.ForEachTask.Task.GenAiComputeTask
	}
	return nil, nil
}

// isAirRun reports whether a run is an AI runtime workload: an ai_runtime_task,
// or a legacy gen_ai_compute_task with a training script.
func isAirRun(r *jobRun) bool {
	ai, gen := r.airTask()
	return ai != nil || (gen != nil && gen.TrainingScriptPath != "")
}

// isSweep reports whether the run's first task fans out into iterations.
func isSweep(r *jobRun) bool {
	return len(r.Tasks) > 0 && r.Tasks[0].ForEachTask != nil
}

// taskRunID returns the run id of the AIR task, used to fetch its MLflow output.
func taskRunID(r *jobRun) int64 {
	if len(r.Tasks) == 0 {
		return 0
	}
	t := r.Tasks[0]
	if t.ForEachTask != nil {
		return t.ForEachTask.Task.RunID
	}
	return t.RunID
}

// jobExperiment returns the run's MLflow experiment name (user-folder prefix
// stripped), or "" when there is none.
func jobExperiment(r *jobRun) string {
	ai, gen := r.airTask()
	switch {
	case ai != nil && ai.Experiment != "":
		return stripExperimentUserPrefix(ai.Experiment)
	case gen != nil && gen.MlflowExperimentName != "":
		return stripExperimentUserPrefix(gen.MlflowExperimentName)
	}
	return ""
}

// jobCompute returns the run's accelerator type and count, or ("", 0) when it
// has none.
func jobCompute(r *jobRun) (string, int) {
	ai, gen := r.airTask()
	switch {
	case ai != nil && len(ai.Deployments) > 0:
		c := ai.Deployments[0].Compute
		return c.AcceleratorType, c.AcceleratorCount
	case gen != nil && gen.Compute != nil:
		return gen.Compute.GpuType, gen.Compute.NumGpus
	}
	return "", 0
}

// jobTiming returns the run's start and end times (epoch ms), preferring the
// first task's window so a run reports its task attempt rather than the wrapper.
func jobTiming(r *jobRun) (startMillis, endMillis int64) {
	startMillis, endMillis = r.StartTime, r.EndTime
	if len(r.Tasks) > 0 {
		if t := r.Tasks[0]; t.StartTime > 0 {
			startMillis = t.StartTime
			endMillis = t.EndTime
		}
	}
	return startMillis, endMillis
}

// fetchJobRunsPage fetches one page of Jobs runs/list. query carries the request
// params (and page_token across calls).
func fetchJobRunsPage(ctx context.Context, w *databricks.WorkspaceClient, query map[string]any) (*jobsRunsListResponse, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	var resp jobsRunsListResponse
	err = apiClient.Do(ctx, http.MethodGet, jobsRunsListPath, nil, nil, query, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}
	return &resp, nil
}
