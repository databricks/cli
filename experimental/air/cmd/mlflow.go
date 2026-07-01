package aircmd

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// getRunOutputResponse is the slice of the jobs runs/get-output response we care
// about. The MLflow identifiers live under ai_runtime_task_output (current) or
// gen_ai_compute_output.run_info (legacy), neither modeled by the typed SDK, so
// we call the endpoint directly and parse just these fields.
type getRunOutputResponse struct {
	AiRuntimeTaskOutput *struct {
		MlflowExperimentID string `json:"mlflow_experiment_id"`
		MlflowRunID        string `json:"mlflow_run_id"`
	} `json:"ai_runtime_task_output"`
	GenAiComputeOutput *struct {
		RunInfo *struct {
			MlflowExperimentID string `json:"mlflow_experiment_id"`
			MlflowRunID        string `json:"mlflow_run_id"`
		} `json:"run_info"`
	} `json:"gen_ai_compute_output"`
}

// mlflowIdentifiers are the experiment and run IDs MLflow assigns to a run.
type mlflowIdentifiers struct {
	ExperimentID string
	RunID        string
}

// mlflowIDs fetches the MLflow IDs for a run via its latest task. Returns nil if
// they can't be obtained.
func mlflowIDs(ctx context.Context, w *databricks.WorkspaceClient, run *jobs.Run) *mlflowIdentifiers {
	if len(run.Tasks) == 0 {
		return nil
	}
	// The MLflow output is attached to the task run, not the parent job run.
	return mlflowIDsForTask(ctx, w, run.Tasks[len(run.Tasks)-1].RunId)
}

// mlflowIDsForTask fetches a task run's MLflow experiment and run IDs from
// runs/get-output, or nil if they can't be obtained. They drive a convenience
// link, so any failure (endpoint error, run not yet started, no MLflow output)
// is logged and treated as "no link" rather than failing the command.
func mlflowIDsForTask(ctx context.Context, w *databricks.WorkspaceClient, taskRunID int64) *mlflowIdentifiers {
	if taskRunID == 0 {
		return nil
	}

	apiClient, err := client.New(w.Config)
	if err != nil {
		log.Debugf(ctx, "air: could not build API client for MLflow link: %v", err)
		return nil
	}

	// Pass run_id through the request arg (the SDK serializes it to the query
	// string for GET); passing it via queryParams instead leaves a nil body that
	// this endpoint rejects with "expected a map".
	var out getRunOutputResponse
	err = apiClient.Do(ctx, http.MethodGet, "/api/2.2/jobs/runs/get-output",
		nil, nil, map[string]any{"run_id": taskRunID}, &out)
	if err != nil {
		log.Debugf(ctx, "air: could not fetch run output for MLflow link: %v", err)
		return nil
	}

	if o := out.AiRuntimeTaskOutput; o != nil && o.MlflowExperimentID != "" && o.MlflowRunID != "" {
		return &mlflowIdentifiers{ExperimentID: o.MlflowExperimentID, RunID: o.MlflowRunID}
	}
	if o := out.GenAiComputeOutput; o != nil && o.RunInfo != nil &&
		o.RunInfo.MlflowExperimentID != "" && o.RunInfo.MlflowRunID != "" {
		return &mlflowIdentifiers{ExperimentID: o.RunInfo.MlflowExperimentID, RunID: o.RunInfo.MlflowRunID}
	}
	return nil
}

// mlflowLogsURL is the deep link to a run's node-0 logs. It is the value of the
// JSON `mlflow_url` field, matching the Python CLI.
func mlflowLogsURL(host string, ids *mlflowIdentifiers) string {
	return fmt.Sprintf("%s/ml/experiments/%s/runs/%s/artifacts/logs/node_0",
		strings.TrimRight(host, "/"), ids.ExperimentID, ids.RunID)
}

// mlflowExperimentURL links to the MLflow experiment page; mlflowRunURL links to
// the run page. These back the Experiment and MLflow Run hyperlinks in text mode.
func mlflowExperimentURL(host string, ids *mlflowIdentifiers) string {
	return fmt.Sprintf("%s/ml/experiments/%s", strings.TrimRight(host, "/"), ids.ExperimentID)
}

func mlflowRunURL(host string, ids *mlflowIdentifiers) string {
	return fmt.Sprintf("%s/ml/experiments/%s/runs/%s",
		strings.TrimRight(host, "/"), ids.ExperimentID, ids.RunID)
}

// mlflowRunLabel returns the MLflow run's human-readable name to use as the
// hyperlink text, falling back to "...{last 8 of run id}" when the name can't be
// fetched. Mirrors Python's _get_mlflow_run_name (cli_display.py).
func mlflowRunLabel(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string) string {
	if name := fetchMLflowRunName(ctx, w, mlflowRunID); name != "" {
		return name
	}
	if len(mlflowRunID) > 8 {
		return "..." + mlflowRunID[len(mlflowRunID)-8:]
	}
	return "..." + mlflowRunID
}

// fetchMLflowRunName fetches a run's MLflow run_name via the MLflow REST API,
// returning "" if it can't be obtained. Best-effort, like the rest of the
// MLflow enrichment.
func fetchMLflowRunName(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string) string {
	apiClient, err := client.New(w.Config)
	if err != nil {
		log.Debugf(ctx, "air get: could not build API client for MLflow run name: %v", err)
		return ""
	}
	var out struct {
		Run struct {
			Info struct {
				RunName string `json:"run_name"`
			} `json:"info"`
		} `json:"run"`
	}
	err = apiClient.Do(ctx, http.MethodGet, "/api/2.0/mlflow/runs/get",
		nil, nil, map[string]any{"run_id": mlflowRunID}, &out)
	if err != nil {
		log.Debugf(ctx, "air get: could not fetch MLflow run name: %v", err)
		return ""
	}
	return out.Run.Info.RunName
}
