package terraform

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"golang.org/x/sync/errgroup"
)

type ErrResourceIsRunning struct {
	resourceType string
	resourceId   string
}

func (e ErrResourceIsRunning) Error() string {
	return fmt.Sprintf("%s %s is running", e.resourceType, e.resourceId)
}

type checkRunningResources struct{}

func (l *checkRunningResources) Name() string {
	return "check-running-resources"
}

func (l *checkRunningResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if !b.Config.Bundle.Deployment.FailOnActiveRuns {
		return nil
	}

	state, err := ParseResourcesState(ctx, b)
	if err != nil && state == nil {
		return diag.FromErr(err)
	}

	w := b.WorkspaceClient()
	err = checkAnyResourceRunning(ctx, w, state)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func CheckRunningResource() *checkRunningResources {
	return &checkRunningResources{}
}

func checkAnyResourceRunning(ctx context.Context, w *databricks.WorkspaceClient, state ExportedResourcesMap) error {
	errs, errCtx := errgroup.WithContext(ctx)

	for _, jobAttrs := range state["jobs"] {
		id := jobAttrs.ID
		if id == "" {
			continue
		}

		errs.Go(func() error {
			isRunning, err := IsJobRunning(errCtx, w, id)
			// If there's an error retrieving the job, we assume it's not running
			if err != nil {
				return err
			}
			if isRunning {
				return &ErrResourceIsRunning{resourceType: "job", resourceId: id}
			}
			return nil
		})
	}

	for _, pipelineAttrs := range state["pipelines"] {
		id := pipelineAttrs.ID
		if id == "" {
			continue
		}

		errs.Go(func() error {
			isRunning, err := IsPipelineRunning(errCtx, w, id)
			// If there's an error retrieving the pipeline, we assume it's not running
			if err != nil {
				return nil
			}
			if isRunning {
				return &ErrResourceIsRunning{resourceType: "pipeline", resourceId: id}
			}
			return nil
		})
	}

	return errs.Wait()
}

func IsJobRunning(ctx context.Context, w *databricks.WorkspaceClient, jobId string) (bool, error) {
	id, err := strconv.Atoi(jobId)
	if err != nil {
		return false, err
	}

	runs, err := w.Jobs.ListRunsAll(ctx, jobs.ListRunsRequest{JobId: int64(id), ActiveOnly: true})
	if err != nil {
		return false, err
	}

	return len(runs) > 0, nil
}

func IsPipelineRunning(ctx context.Context, w *databricks.WorkspaceClient, pipelineId string) (bool, error) {
	resp, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{PipelineId: pipelineId})
	if err != nil {
		return false, err
	}
	switch resp.State {
	case pipelines.PipelineStateIdle, pipelines.PipelineStateFailed, pipelines.PipelineStateDeleted:
		return false, nil
	default:
		return true, nil
	}
}
