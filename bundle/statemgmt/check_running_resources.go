package statemgmt

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
	sdkpipelines "github.com/databricks/sdk-go/pipelines/v2"
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

	w := b.WorkspaceClient(ctx)
	err := checkAnyResourceRunning(ctx, w, b.DeploymentBundle.ExportState(ctx))
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func CheckRunningResource() bundle.Mutator {
	return &checkRunningResources{}
}

func checkAnyResourceRunning(ctx context.Context, w *sdkclients.Clients, state ExportedResourcesMap) error {
	errs, errCtx := errgroup.WithContext(ctx)

	for resourceKey, attrs := range state {
		id := attrs.ID
		if id == "" {
			continue
		}

		resourceType := config.GetResourceTypeFromKey(resourceKey)

		if resourceType == "jobs" {
			errs.Go(func() error {
				isRunning, err := IsJobRunning(errCtx, w, id)
				if err != nil {
					return err
				}
				if isRunning {
					return &ErrResourceIsRunning{resourceType: "job", resourceId: id}
				}
				return nil
			})
		}

		if resourceType == "pipelines" {
			errs.Go(func() error {
				isRunning, err := IsPipelineRunning(errCtx, w, id)
				if err != nil {
					return err
				}
				if isRunning {
					return &ErrResourceIsRunning{resourceType: "pipeline", resourceId: id}
				}
				return nil
			})
		}
	}

	return errs.Wait()
}

func IsJobRunning(ctx context.Context, w *sdkclients.Clients, jobId string) (bool, error) {
	id, err := strconv.Atoi(jobId)
	if err != nil {
		return false, err
	}

	for _, err := range w.Jobs.ListRunsIter(ctx, sdkjobs.ListRunsRequest{
		JobId:           new(int64(id)),
		StateConstraint: &sdkjobs.ListRunsRequest_StateConstraint_ActiveOnly{ActiveOnly: true},
	}) {
		if apierrs.IsMissing(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		// Any active run means the job is running.
		return true, nil
	}

	return false, nil
}

func IsPipelineRunning(ctx context.Context, w *sdkclients.Clients, pipelineId string) (bool, error) {
	resp, err := w.Pipelines.Get(ctx, sdkpipelines.GetPipelineRequest{PipelineId: new(pipelineId)})
	if apierrs.IsMissing(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch resp.State {
	case pipelines.PipelineState_PipelineState_Idle, pipelines.PipelineState_PipelineState_Failed, pipelines.PipelineState_PipelineState_Deleted:
		return false, nil
	default:
		return true, nil
	}
}
