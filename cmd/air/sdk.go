package aircmd

import (
	"context"
	"iter"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/sdkconv"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
)

// The air commands operate on the JSON mirrors of the Jobs types (they render
// runs as JSON and build submit bodies from them); these helpers convert at
// the API boundary.

// getRun fetches a run with all its tasks. runs/get returns at most 100 tasks
// per page, so the remaining pages are fetched and merged.
func getRun(ctx context.Context, w *sdkclients.Clients, runID int64) (*jobs.GetRunResponse, error) {
	resp, err := w.Jobs.GetRun(ctx, sdkjobs.GetRunRequest{RunId: new(runID)})
	if err != nil {
		return nil, err
	}
	run, err := sdkconv.FromSDK[jobs.GetRunResponse](resp)
	if err != nil {
		return nil, err
	}
	for run.NextPageToken != nil && *run.NextPageToken != "" {
		resp, err := w.Jobs.GetRun(ctx, sdkjobs.GetRunRequest{RunId: new(runID), PageToken: run.NextPageToken})
		if err != nil {
			return nil, err
		}
		next, err := sdkconv.FromSDK[jobs.GetRunResponse](resp)
		if err != nil {
			return nil, err
		}
		run.Tasks = append(run.Tasks, next.Tasks...)
		run.JobClusters = append(run.JobClusters, next.JobClusters...)
		run.JobParameters = append(run.JobParameters, next.JobParameters...)
		run.RepairHistory = append(run.RepairHistory, next.RepairHistory...)
		run.NextPageToken = next.NextPageToken
	}
	return run, nil
}

// getRunOutput fetches the output of a single task run.
func getRunOutput(ctx context.Context, w *sdkclients.Clients, taskRunID int64) (*jobs.GetRunOutputResponse, error) {
	resp, err := w.Jobs.GetRunOutput(ctx, sdkjobs.GetRunOutputRequest{RunId: new(taskRunID)})
	if err != nil {
		return nil, err
	}
	return sdkconv.FromSDK[jobs.GetRunOutputResponse](resp)
}

// listRuns yields the runs matching req, converted to mirrors.
func listRuns(ctx context.Context, w *sdkclients.Clients, req sdkjobs.ListRunsRequest) iter.Seq2[*jobs.BaseRun, error] {
	return func(yield func(*jobs.BaseRun, error) bool) {
		for base, err := range w.Jobs.ListRunsIter(ctx, req) {
			if err != nil {
				yield(nil, err)
				return
			}
			run, err := sdkconv.FromSDK[jobs.BaseRun](base)
			if !yield(run, err) || err != nil {
				return
			}
		}
	}
}

// currentUserName returns the user name of the authenticated principal.
func currentUserName(ctx context.Context, w *sdkclients.Clients) (string, error) {
	me, err := w.Scim.Me(ctx, sdkscim.MeRequest{})
	if err != nil {
		return "", err
	}
	if me.UserName == nil {
		return "", nil
	}
	return *me.UserName, nil
}
