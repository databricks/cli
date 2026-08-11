package aircmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// mlflowLinkPollAttempts bounds the best-effort poll for a freshly-submitted
// run's MLflow IDs (see resolveMLflowIDsForRun), kept short so a bare `air run`
// returns promptly when the IDs aren't ready yet.
const mlflowLinkPollAttempts = 3

// mlflowLinkPollInterval is the delay between poll attempts. A var, not a const,
// so tests can shrink it and avoid a real sleep.
var mlflowLinkPollInterval = 500 * time.Millisecond

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

	out, err := w.Jobs.GetRunOutputByRunId(ctx, taskRunID)
	if err != nil {
		log.Debugf(ctx, "air: could not fetch run output for MLflow link: %v", err)
		return nil
	}

	if o := out.AiRuntimeTaskOutput; o != nil && o.MlflowExperimentId != "" && o.MlflowRunId != "" {
		return &mlflowIdentifiers{ExperimentID: o.MlflowExperimentId, RunID: o.MlflowRunId}
	}
	return nil
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

// mlflowExperimentURL links to the MLflow experiment page. Mirrors Python's
// mlflow_experiment_url (cli_display.py); like the other run-submit links it
// omits the ?o= workspace-id query the Python CLI adds, matching mlflowRunURL
// and the run-submit dashboard URL.
func mlflowExperimentURL(host string, ids *mlflowIdentifiers) string {
	return fmt.Sprintf("%s/ml/experiments/%s", strings.TrimRight(host, "/"), ids.ExperimentID)
}

// resolveMLflowIDsForRun best-effort resolves a just-submitted run's MLflow
// experiment/run IDs. They are assigned only once the task run starts, so it
// polls runs/get + runs/get-output a few times (mirroring the Python CLI, which
// polls for the same IDs after submit — cli_entrypoint.py _try_update_mlflow_run_name).
// Returns nil if they don't appear within the budget or the context is
// cancelled; callers treat that as "no MLflow link" rather than an error, since
// the run is already submitted.
func resolveMLflowIDsForRun(ctx context.Context, w *databricks.WorkspaceClient, runID int64) *mlflowIdentifiers {
	// The task run id is fixed at submit time, so resolve it once; only the MLflow
	// output (runs/get-output) fills in later, so that is all we poll.
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		log.Debugf(ctx, "air run: could not fetch run %d for MLflow link: %v", runID, err)
		return nil
	}
	if len(run.Tasks) == 0 {
		return nil
	}
	// The MLflow output is attached to the task run, not the parent job run.
	taskRunID := run.Tasks[len(run.Tasks)-1].RunId

	for attempt := range mlflowLinkPollAttempts {
		if ids := mlflowIDsForTask(ctx, w, taskRunID); ids != nil {
			return ids
		}
		if attempt == mlflowLinkPollAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(mlflowLinkPollInterval):
		}
	}
	return nil
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
