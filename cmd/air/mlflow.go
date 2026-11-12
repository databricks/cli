package aircmd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// mlflowIdentifiers are the experiment and run IDs MLflow assigns to a run.
type mlflowIdentifiers struct {
	ExperimentID string
	RunID        string
}

// mlflowIDs fetches the MLflow IDs a run's views link to (see
// mlflowIDsFromRetries). Returns nil if they can't be obtained.
func mlflowIDs(ctx context.Context, w *databricks.WorkspaceClient, run *jobs.Run) *mlflowIdentifiers {
	latest, err := latestTaskOutput(ctx, w, run.Tasks)
	return mlflowIDsFromRetries(ctx, w, run.Tasks, isTerminal(run), latest, err)
}

// mlflowIDsFromRetries returns the IDs from latest (the latest retry's output),
// else, for a finished run, from the newest earlier retry of the same task that
// has them: a last retry that failed before it started never got an MLflow run
// of its own. A run still in progress keeps only the latest retry's IDs, so its
// link and ETA never point at a previous retry.
//
// If a get-output call fails (latestErr, or an earlier retry's), it returns nil
// rather than skip that retry: the retry it couldn't read may have the newest
// MLflow run, and an older link would then look current.
func mlflowIDsFromRetries(ctx context.Context, w *databricks.WorkspaceClient, tasks []jobs.RunTask, terminal bool, latest *jobs.AiRuntimeTaskOutput, latestErr error) *mlflowIdentifiers {
	if latestErr != nil {
		return nil
	}
	if ids := mlflowIDsFromOutput(latest); ids != nil || len(tasks) < 2 || !terminal {
		return ids
	}
	last := tasks[len(tasks)-1]
	for _, task := range slices.Backward(tasks[:len(tasks)-1]) {
		if task.TaskKey != last.TaskKey {
			continue
		}
		out, err := aiRuntimeTaskOutputForTask(ctx, w, task.RunId)
		if err != nil {
			return nil
		}
		if ids := mlflowIDsFromOutput(out); ids != nil {
			return ids
		}
	}
	return nil
}

func aiRuntimeTaskOutput(ctx context.Context, w *databricks.WorkspaceClient, run *jobs.Run) *jobs.AiRuntimeTaskOutput {
	out, _ := latestTaskOutput(ctx, w, run.Tasks)
	return out
}

// latestTaskOutput fetches the output of the last of tasks, the run's latest
// retry (see aiRuntimeTaskOutputForTask).
func latestTaskOutput(ctx context.Context, w *databricks.WorkspaceClient, tasks []jobs.RunTask) (*jobs.AiRuntimeTaskOutput, error) {
	if len(tasks) == 0 {
		return nil, nil
	}
	return aiRuntimeTaskOutputForTask(ctx, w, tasks[len(tasks)-1].RunId)
}

// mlflowIDsForTask fetches a task run's MLflow experiment and run IDs from
// runs/get-output, or nil if they can't be obtained. They drive a convenience
// link, so any failure (endpoint error, run not yet started, no MLflow output)
// is logged and treated as "no link" rather than failing the command.
func mlflowIDsForTask(ctx context.Context, w *databricks.WorkspaceClient, taskRunID int64) *mlflowIdentifiers {
	out, _ := aiRuntimeTaskOutputForTask(ctx, w, taskRunID)
	return mlflowIDsFromOutput(out)
}

// aiRuntimeTaskOutputForTask fetches a task run's AI runtime output, logging
// any fetch error before returning it.
func aiRuntimeTaskOutputForTask(ctx context.Context, w *databricks.WorkspaceClient, taskRunID int64) (*jobs.AiRuntimeTaskOutput, error) {
	if taskRunID == 0 {
		return nil, nil
	}

	out, err := w.Jobs.GetRunOutputByRunId(ctx, taskRunID)
	if err != nil {
		log.Debugf(ctx, "air: could not fetch run output for MLflow link: %v", err)
		return nil, err
	}

	return out.AiRuntimeTaskOutput, nil
}

func mlflowIDsFromOutput(output *jobs.AiRuntimeTaskOutput) *mlflowIdentifiers {
	if output == nil || output.MlflowExperimentId == "" || output.MlflowRunId == "" {
		return nil
	}
	return &mlflowIdentifiers{ExperimentID: output.MlflowExperimentId, RunID: output.MlflowRunId}
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
func fetchMLflowRunName(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string) string {
	resp, err := w.Experiments.GetRun(ctx, ml.GetRunRequest{RunId: mlflowRunID})
	if err != nil {
		log.Debugf(ctx, "air get: could not fetch MLflow run name: %v", err)
		return ""
	}
	if resp.Run == nil || resp.Run.Info == nil {
		return ""
	}
	return resp.Run.Info.RunName
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
