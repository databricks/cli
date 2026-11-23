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
	run, data, err := taskView(run, taskKey)
	if err != nil {
		return "", err
	}

	// As in `air get`: an ai_runtime_task run keeps its experiment, accelerators
	// and config path on the task.
	if genAIComputeTask(run) == nil {
		enrichFromAiRuntimeTask(run, &data)
	}
	data.DashboardURL = dashboardURL(w.Config.Host, runID, workspaceIDForRunLink(ctx, w))
	if findForEachTask(run) == nil && genAIComputeTask(run) == nil {
		enrichFromAiRuntimeTask(run, &data)
	}
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
// list picker's detail pane, followed by the run's MLflow logs link. taskKey is
// the selected row's task; the pane notes when the logs belong to another task
// (see logTaskNote).
func runLogsSnapshot(ctx context.Context, w *databricks.WorkspaceClient, runID int64, taskKey string) (string, error) {
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
	if note := logTaskNote(run, taskKey); note != "" {
		fmt.Fprintln(&buf, note)
		fmt.Fprintln(&buf)
	}
	if _, err := fetchLogs(ctx, w, &buf, req, status); err != nil {
		return "", fmt.Errorf("failed to fetch logs: %w", err)
	}
	if ids := mlflowIDs(ctx, w, run); ids != nil {
		fmt.Fprintf(&buf, "\nMLflow URL%s: %s\n", ids.retryNote(), mlflowLogsURL(w.Config.Host, ids))
	}
	return buf.String(), nil
}

// logTaskNote returns a note for the logs pane when the logs it shows are not
// those of taskKey, the selected row's task: logs are read at run level from
// the run's latest task (see resolveLogAttempt), so on a run with several AI
// runtime tasks they can belong to another one. It returns "" when they match
// or no task was selected.
func logTaskNote(run *jobs.Run, taskKey string) string {
	if taskKey == "" {
		return ""
	}
	_, taskRunID, err := resolveLogAttempt(run, -1)
	if err != nil {
		return ""
	}
	for _, t := range run.Tasks {
		if t.RunId == taskRunID && t.TaskKey != taskKey {
			return fmt.Sprintf("Note: these are the logs of task %q, the run's latest task, not of the selected task %q. Logs are not yet available per task.", t.TaskKey, taskKey)
		}
	}
	return ""
}
