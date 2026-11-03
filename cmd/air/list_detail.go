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
// a string, for the list picker's detail pane.
func runDetailText(ctx context.Context, w *databricks.WorkspaceClient, runID int64) (string, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		if errors.Is(err, apierr.ErrResourceDoesNotExist) {
			return "", fmt.Errorf("run %d not found", runID)
		}
		return "", fmt.Errorf("failed to fetch run: %w", err)
	}

	// A missing workspace id only drops the ?o= org hint from the dashboard link.
	workspaceID, _ := w.CurrentWorkspaceID(ctx)

	data := buildGetData(run)
	data.DashboardURL = dashboardURL(w.Config.Host, runID, workspaceID)
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
// list picker's detail pane.
func runLogsSnapshot(ctx context.Context, w *databricks.WorkspaceClient, runID int64) (string, error) {
	status, err := resolveRunStatus(ctx, w, runID)
	if err != nil {
		if errors.Is(err, apierr.ErrResourceDoesNotExist) {
			return "", fmt.Errorf("run %d not found", runID)
		}
		return "", fmt.Errorf("failed to fetch run status: %w", err)
	}

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
	return buf.String(), nil
}
