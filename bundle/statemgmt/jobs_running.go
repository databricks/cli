package statemgmt

import (
	"context"
	"strconv"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func init() {
	runningChecks.Register("jobs", IsJobRunning)
}

func IsJobRunning(ctx context.Context, w *databricks.WorkspaceClient, jobId string) (bool, error) {
	id, err := strconv.Atoi(jobId)
	if err != nil {
		return false, err
	}

	runs, err := w.Jobs.ListRunsAll(ctx, jobs.ListRunsRequest{JobId: int64(id), ActiveOnly: true})
	if apierr.IsMissing(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return len(runs) > 0, nil
}
