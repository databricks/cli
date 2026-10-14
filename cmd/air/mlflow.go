package aircmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/experiments/v1"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
)

// mlflowIdentifiers are the experiment and run IDs MLflow assigns to a run.
type mlflowIdentifiers struct {
	ExperimentID string
	RunID        string
}

// mlflowIDs fetches the MLflow IDs for a run via its latest task. Returns nil if
// they can't be obtained.
func mlflowIDs(ctx context.Context, w *sdkclients.Clients, run *jobs.GetRunResponse) *mlflowIdentifiers {
	return mlflowIDsFromOutput(aiRuntimeTaskOutput(ctx, w, run))
}

func aiRuntimeTaskOutput(ctx context.Context, w *sdkclients.Clients, run *jobs.GetRunResponse) *jobs.AiRuntimeTaskOutput {
	if len(run.Tasks) == 0 {
		return nil
	}
	return aiRuntimeTaskOutputForTask(ctx, w, run.Tasks[len(run.Tasks)-1].RunId)
}

// mlflowIDsForTask fetches a task run's MLflow experiment and run IDs from
// runs/get-output, or nil if they can't be obtained. They drive a convenience
// link, so any failure (endpoint error, run not yet started, no MLflow output)
// is logged and treated as "no link" rather than failing the command.
func mlflowIDsForTask(ctx context.Context, w *sdkclients.Clients, taskRunID int64) *mlflowIdentifiers {
	return mlflowIDsFromOutput(aiRuntimeTaskOutputForTask(ctx, w, taskRunID))
}

func aiRuntimeTaskOutputForTask(ctx context.Context, w *sdkclients.Clients, taskRunID int64) *jobs.AiRuntimeTaskOutput {
	if taskRunID == 0 {
		return nil
	}

	out, err := getRunOutput(ctx, w, taskRunID)
	if err != nil {
		log.Debugf(ctx, "air: could not fetch run output for MLflow link: %v", err)
		return nil
	}

	return out.AiRuntimeTaskOutput
}

func mlflowIDsFromOutput(output *jobs.AiRuntimeTaskOutput) *mlflowIdentifiers {
	if output == nil || utils.Deref(output.MlflowExperimentId) == "" || utils.Deref(output.MlflowRunId) == "" {
		return nil
	}
	return &mlflowIdentifiers{ExperimentID: utils.Deref(output.MlflowExperimentId), RunID: utils.Deref(output.MlflowRunId)}
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

// mlflowExperimentURL links to the MLflow experiment page. Omits the ?o= query
// for consistency with mlflowRunURL and the run-submit dashboard URL.
func mlflowExperimentURL(host string, ids *mlflowIdentifiers) string {
	return fmt.Sprintf("%s/ml/experiments/%s", strings.TrimRight(host, "/"), ids.ExperimentID)
}

// fetchMLflowRunName fetches a run's MLflow run_name via the MLflow REST API,
// returning "" if it can't be obtained. Best-effort, like the rest of the MLflow
// enrichment.
func fetchMLflowRunName(ctx context.Context, w *sdkclients.Clients, mlflowRunID string) string {
	resp, err := w.Experiments.GetRun(ctx, experiments.GetRunRequest{RunId: new(mlflowRunID)})
	if err != nil {
		log.Debugf(ctx, "air get: could not fetch MLflow run name: %v", err)
		return ""
	}
	if resp.Run == nil || resp.Run.Info == nil {
		return ""
	}
	return utils.Deref(resp.Run.Info.RunName)
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
