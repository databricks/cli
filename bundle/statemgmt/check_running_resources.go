package statemgmt

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/terraform"
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

type checkRunningResources struct {
	engine engine.EngineType
}

func (l *checkRunningResources) Name() string {
	return "check-running-resources"
}

func (l *checkRunningResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if !b.Config.Bundle.Deployment.FailOnActiveRuns {
		return nil
	}

	var err error
	var state ExportedResourcesMap

	if l.engine.IsDirect() {
		_, fullPathDirect := b.StateFilenameDirect(ctx)
		state, err = b.DeploymentBundle.ExportState(ctx, fullPathDirect)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		state, err = terraform.ParseResourcesState(ctx, b)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	w := b.WorkspaceClient()
	err = checkAnyResourceRunning(ctx, w, state)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func CheckRunningResource(engine engine.EngineType) bundle.Mutator {
	return &checkRunningResources{engine: engine}
}

func checkAnyResourceRunning(ctx context.Context, w *databricks.WorkspaceClient, state ExportedResourcesMap) error {
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

		if resourceType == "pipelines" {
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
