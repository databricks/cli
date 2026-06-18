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
// about. The MLflow identifiers live under a gen_ai_compute_output field that
// the typed SDK does not model, so we call the endpoint directly and parse just
// these fields.
type getRunOutputResponse struct {
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

// mlflowIDs fetches the run's MLflow experiment and run IDs, or nil if they
// can't be obtained. They drive a convenience link, so any failure here
// (missing task, endpoint error, run not yet started) is logged and treated as
// "no link" rather than failing the whole command.
func mlflowIDs(ctx context.Context, w *databricks.WorkspaceClient, run *jobs.Run) *mlflowIdentifiers {
	if len(run.Tasks) == 0 {
		return nil
	}
	// The MLflow output is attached to the task run, not the parent job run.
	taskRunID := run.Tasks[len(run.Tasks)-1].RunId

	apiClient, err := client.New(w.Config)
	if err != nil {
		log.Debugf(ctx, "air get: could not build API client for MLflow link: %v", err)
		return nil
	}

	// Pass run_id through the request arg (the SDK serializes it to the query
	// string for GET); passing it via queryParams instead leaves a nil body that
	// this endpoint rejects with "expected a map".
	var out getRunOutputResponse
	err = apiClient.Do(ctx, http.MethodGet, "/api/2.2/jobs/runs/get-output",
		nil, nil, map[string]any{"run_id": taskRunID}, &out)
	if err != nil {
		log.Debugf(ctx, "air get: could not fetch run output for MLflow link: %v", err)
		return nil
	}

	if out.GenAiComputeOutput == nil || out.GenAiComputeOutput.RunInfo == nil {
		return nil
	}
	info := out.GenAiComputeOutput.RunInfo
	if info.MlflowExperimentID == "" || info.MlflowRunID == "" {
		return nil
	}
	return &mlflowIdentifiers{ExperimentID: info.MlflowExperimentID, RunID: info.MlflowRunID}
}

// mlflowLogsURL is the deep link to a run's node-0 logs. It is the value of the
// JSON `mlflow_url` field, matching the Python CLI.
func mlflowLogsURL(host string, ids *mlflowIdentifiers) string {
	return fmt.Sprintf("%s/ml/experiments/%s/runs/%s/artifacts/logs/node_0",
		strings.TrimRight(host, "/"), ids.ExperimentID, ids.RunID)
}

// mlflowRunURL links to the MLflow run page; it backs the MLflow Run hyperlink
// in the single-run view.
func mlflowRunURL(host string, ids *mlflowIdentifiers) string {
	return fmt.Sprintf("%s/ml/experiments/%s/runs/%s",
		strings.TrimRight(host, "/"), ids.ExperimentID, ids.RunID)
}

// fetchMLflowRunName fetches a run's MLflow run_name via the MLflow REST API,
// returning "" if it can't be obtained. Best-effort, like the rest of the MLflow
// enrichment.
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

// mlflowRunLabel is the text shown for the MLflow Run cell: the run's name, or
// "...{last 8 of run id}" when the name is unknown. Mirrors Python's
// _get_mlflow_run_name (cli_display.py).
func mlflowRunLabel(name, mlflowRunID string) string {
	if name != "" {
		return name
	}
	if len(mlflowRunID) > 8 {
		return "..." + mlflowRunID[len(mlflowRunID)-8:]
	}
	return "..." + mlflowRunID
}
