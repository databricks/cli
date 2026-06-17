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

// mlflowURL returns a link to the run's MLflow logs, or nil if it can't be
// built. The link is a convenience, so any failure here (missing task, endpoint
// error, run not yet started) is logged and treated as "no link" rather than
// failing the whole command.
func mlflowURL(ctx context.Context, w *databricks.WorkspaceClient, run *jobs.Run) *string {
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

	host := strings.TrimRight(w.Config.Host, "/")
	url := fmt.Sprintf("%s/ml/experiments/%s/runs/%s/artifacts/logs/node_0",
		host, info.MlflowExperimentID, info.MlflowRunID)
	return &url
}
