package aircmd

import (
	"context"
	"iter"
	"net/http"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
)

// getRun fetches a run with all its tasks. runs/get returns at most 100 tasks
// per page, so the remaining pages are fetched and merged.
func getRun(ctx context.Context, w *sdkclients.Clients, runID int64) (*jobs.GetRunResponse, error) {
	resp, err := w.Jobs.GetRun(ctx, sdkjobs.GetRunRequest{RunId: new(runID)})
	if err != nil {
		return nil, err
	}
	run, err := jobs.GetRunResponseFromSDK(resp)
	if err != nil {
		return nil, err
	}
	for run.NextPageToken != nil && *run.NextPageToken != "" {
		resp, err := w.Jobs.GetRun(ctx, sdkjobs.GetRunRequest{RunId: new(runID), PageToken: run.NextPageToken})
		if err != nil {
			return nil, err
		}
		next, err := jobs.GetRunResponseFromSDK(resp)
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
	return jobs.GetRunOutputResponseFromSDK(resp)
}

// listRuns yields the runs matching req, converted to mirrors.
func listRuns(ctx context.Context, w *sdkclients.Clients, req sdkjobs.ListRunsRequest) iter.Seq2[*jobs.BaseRun, error] {
	return func(yield func(*jobs.BaseRun, error) bool) {
		for base, err := range w.Jobs.ListRunsIter(ctx, req) {
			if err != nil {
				yield(nil, err)
				return
			}
			run, err := jobs.BaseRunFromSDK(base)
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

// getRunRaw decodes the runs/get response for runID into out. It is used for
// response fields sdk-go does not model.
func getRunRaw(ctx context.Context, w *sdkclients.Clients, runID int64, out any) error {
	apiClient, err := apiclient.New(w.Config)
	if err != nil {
		return err
	}
	query := map[string]any{"run_id": runID}
	return apiClient.Do(ctx, http.MethodGet, jobsRunsGetPath, auth.WorkspaceIDHeaders(w.Config), query, nil, out)
}
