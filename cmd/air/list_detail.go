package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// runDetailText fetches a run and renders the same styled view as `air get` into
// a string, for the list picker's detail pane. taskKey selects the AI runtime
// task shown, as `air get --task-key` does; "" means the first.
func runDetailText(ctx context.Context, w *databricks.WorkspaceClient, runID int64, taskKey string) (string, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		if errors.Is(err, apierr.ErrResourceDoesNotExist) {
			return "", fmt.Errorf("run %d not found", runID)
		}
		return "", fmt.Errorf("failed to fetch run: %w", err)
	}
	fullRun := run
	run, err = selectAirTask(run, taskKey)
	if err != nil {
		return "", err
	}

	data := buildGetData(run)
	annotateTask(fullRun, run, &data)
	// As in `air get`: an ai_runtime_task run keeps its experiment, accelerators
	// and config path on the task.
	if genAIComputeTask(run) == nil {
		enrichFromAiRuntimeTask(run, &data)
	}
	data.DashboardURL = dashboardURL(w.Config.Host, runID, workspaceIDForRunLink(ctx, w))
	ids := mlflowIDs(ctx, w, run)
	if ids != nil {
		url := mlflowLogsURL(w.Config.Host, ids)
		data.MLflowURL = &url
	}

	var buf bytes.Buffer
	renderRunText(ctx, &buf, w, run, &data, ids)
	return buf.String(), nil
}

// runLogsSnapshot fetches a one-shot tail of a run's logs into a string, for the
// list picker's detail pane, followed by the run's MLflow logs link.
func runLogsSnapshot(ctx context.Context, w *databricks.WorkspaceClient, runID int64) (string, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		if errors.Is(err, apierr.ErrResourceDoesNotExist) {
			return "", fmt.Errorf("run %d not found", runID)
		}
		return "", fmt.Errorf("failed to fetch run status: %w", err)
	}
	status := projectRunStatus(run)

	req := logRequest{
		runID:      runID,
		attempt:    -1,   // latest attempt
		tailLines:  1000, // one-shot tail, not a live follow
		staticView: true,
	}

	var buf bytes.Buffer
	if _, err := fetchLogs(ctx, w, &buf, req, status); err != nil {
		return "", fmt.Errorf("failed to fetch logs: %w", err)
	}
	if ids := mlflowIDs(ctx, w, run); ids != nil {
		fmt.Fprintf(&buf, "\nMLflow URL%s: %s\n", ids.retryNote(), mlflowLogsURL(w.Config.Host, ids))
	}
	return buf.String(), nil
}
